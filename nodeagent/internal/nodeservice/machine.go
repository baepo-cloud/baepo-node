package nodeservice

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/nodeservice/machinecontroller"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/pbadapter"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log/slog"
)

func (s *Service) loadMachines(ctx context.Context) error {
	s.log.Info("loading machines")

	var machines []*types.Machine
	err := s.db.WithContext(ctx).
		Preload("Volumes.Volume").
		Preload("Volumes.Image.Volume").
		Joins("NetworkInterface").
		Where("machines.state NOT IN ?", []types.MachineState{types.MachineStateTerminated}).
		Find(&machines).
		Error
	if err != nil {
		return fmt.Errorf("failed to retrieve machines: %w", err)
	}

	s.machineControllerLock.Lock()
	defer s.machineControllerLock.Unlock()

	for _, machine := range machines {
		s.machineControllers[machine.ID] = s.newMachineController(machine)
	}

	return nil
}

func (s *Service) newMachineController(machine *types.Machine) *machinecontroller.Controller {
	ctrl := machinecontroller.New(
		s.db, s.volumeProvider, s.networkProvider, s.runtimeProvider, s.imageProvider,
		machine,
	)
	ctrl.SubscribeToEvents(func(ctx context.Context, event *corev1pb.MachineEvent) {
		go func() {
			s.machineEvents <- event
		}()
	})
	return ctrl
}

func (s *Service) FindMachine(ctx context.Context, machineID string) (*types.Machine, error) {
	s.machineControllerLock.RLock()
	defer s.machineControllerLock.RUnlock()

	ctrl, ok := s.machineControllers[machineID]
	if !ok {
		return nil, types.ErrMachineNotFound
	}

	return ctrl.GetMachine(), nil
}

func (s *Service) ListMachines(ctx context.Context) ([]*types.Machine, error) {
	s.machineControllerLock.RLock()
	defer s.machineControllerLock.RUnlock()

	var machines []*types.Machine
	for _, ctrl := range s.machineControllers {
		machines = append(machines, ctrl.GetMachine())
	}
	return machines, nil
}

func (s *Service) CreateMachine(ctx context.Context, opts types.NodeCreateMachineOptions) (*types.Machine, error) {
	s.log.Info("requesting machine creation", slog.String("machine-id", opts.MachineID))

	machine := &types.Machine{
		ID:           opts.MachineID,
		State:        types.MachineStatePending,
		DesiredState: opts.DesiredState,
		Spec:         &opts.Spec,
	}
	if err := s.db.WithContext(ctx).Save(&machine).Error; err != nil {
		return nil, fmt.Errorf("failed to create machine: %w", err)
	}

	s.machineControllerLock.Lock()
	defer s.machineControllerLock.Unlock()
	s.machineControllers[machine.ID] = s.newMachineController(machine)

	return machine, nil
}

func (s *Service) UpdateMachineDesiredState(ctx context.Context, opts types.NodeUpdateMachineDesiredStateOptions) (*types.Machine, error) {
	s.machineControllerLock.RLock()
	defer s.machineControllerLock.RUnlock()

	ctrl, ok := s.machineControllers[opts.MachineID]
	if !ok {
		return nil, types.ErrMachineNotFound
	}

	s.log.Info("requesting new machine desired state",
		slog.String("machine-id", opts.MachineID),
		slog.Any("desired-state", opts.DesiredState))
	ctrl.PublishEvent(&corev1pb.MachineEvent{
		Timestamp: timestamppb.Now(),
		MachineId: opts.MachineID,
		Event: &corev1pb.MachineEvent_DesiredStateChangedEvent{
			DesiredStateChangedEvent: &corev1pb.MachineEvent_DesiredStateChanged{
				DesiredState: pbadapter.MachineDesiredStateToProto(opts.DesiredState),
			},
		},
	})
	return ctrl.GetMachine(), nil
}
