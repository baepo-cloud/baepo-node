package types

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
	"google.golang.org/protobuf/proto"
	"time"
)

type (
	MachineState string

	MachineDesiredState string

	Machine struct {
		ID                 string `gorm:"primaryKey"`
		State              MachineState
		DesiredState       MachineDesiredState
		RuntimePID         *int `gorm:"column:runtime_pid"`
		Spec               *MachineSpec
		NetworkInterfaceID *string
		Volumes            []*MachineVolume
		Containers         []*Container
		NetworkInterface   *NetworkInterface
		CreatedAt          time.Time
		TerminatedAt       *time.Time
	}

	MachineEventType string

	MachineEvent struct {
		ID          string `gorm:"primaryKey"`
		Type        MachineEventType
		MachineID   string
		Machine     *Machine
		ContainerID *string
		Container   *Container
		Payload     []byte
		Timestamp   time.Time
	}

	MachineSpec struct {
		Cpus     uint32
		MemoryMB uint64
	}

	MachineVolume struct {
		ID          string
		Position    int
		MachineID   string
		Machine     *Machine
		ContainerID string
		Container   *Container
		ImageID     *string
		Image       *Image
		VolumeID    string
		Volume      *Volume
		CreatedAt   time.Time
	}

	MachineCreateOptions struct {
		MachineID    string
		DesiredState MachineDesiredState
		Spec         MachineSpec
		Containers   []MachineCreateContainerOptions
	}

	MachineCreateContainerOptions struct {
		ContainerID string
		Spec        ContainerSpec
	}

	MachineUpdateDesiredStateOptions struct {
		MachineID    string
		DesiredState MachineDesiredState
	}

	MachineService interface {
		List(ctx context.Context) ([]*Machine, error)

		FindByID(ctx context.Context, machineID string) (*Machine, error)

		Create(ctx context.Context, opts MachineCreateOptions) (*Machine, error)

		UpdateDesiredState(ctx context.Context, opts MachineUpdateDesiredStateOptions) (*Machine, error)

		ListEvents(ctx context.Context, machineID string) ([]*MachineEvent, error)

		SubscribeToEvents(ctx context.Context) <-chan *MachineEvent
	}
)

const (
	MachineStatePending     MachineState = "pending"
	MachineStateStarting    MachineState = "starting"
	MachineStateRunning     MachineState = "running"
	MachineStateDegraded    MachineState = "degraded"
	MachineStateError       MachineState = "error"
	MachineStateTerminating MachineState = "terminating"
	MachineStateTerminated  MachineState = "terminated"

	MachineDesiredStatePending    MachineDesiredState = "pending"
	MachineDesiredStateRunning    MachineDesiredState = "running"
	MachineDesiredStateTerminated MachineDesiredState = "terminated"

	MachineEventTypeStateChanged          MachineEventType = "state_changed"
	MachineEventTypeDesiredStateChanged   MachineEventType = "desired_state_changed"
	MachineEventTypeContainerStateChanged MachineEventType = "container_state_changed"
)

var ErrMachineNotFound = errors.New("machine not found")

func (*MachineSpec) GormDataType() string {
	return "jsonb"
}

func (s *MachineSpec) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &s)
}

func (s *MachineSpec) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s MachineState) MatchDesiredState(desired MachineDesiredState) bool {
	switch s {
	case MachineStatePending:
		return desired == MachineDesiredStatePending
	case MachineStateRunning, MachineStateDegraded:
		return desired == MachineDesiredStateRunning
	case MachineStateTerminated:
		return desired == MachineDesiredStateTerminated
	default:
		return false
	}
}

func (e MachineEvent) ProtoPayload() (proto.Message, error) {
	switch e.Type {
	case MachineEventTypeContainerStateChanged:
		var event corev1pb.ContainerEvent
		if err := proto.Unmarshal(e.Payload, &event); err != nil {
			return nil, err
		}
		return &event, nil
	case MachineEventTypeDesiredStateChanged, MachineEventTypeStateChanged:
		var event corev1pb.MachineEvent
		if err := proto.Unmarshal(e.Payload, &event); err != nil {
			return nil, err
		}
		return &event, nil
	default:
		return nil, fmt.Errorf("unknown proto type: %v", e.Type)
	}
}
