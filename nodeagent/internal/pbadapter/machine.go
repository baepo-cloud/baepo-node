package pbadapter

import (
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
)

func MachineStateToProto(state types.MachineState) corev1pb.MachineState {
	switch state {
	case types.MachineStatePending:
		return corev1pb.MachineState_MachineState_Pending
	case types.MachineStateStarting:
		return corev1pb.MachineState_MachineState_Starting
	case types.MachineStateRunning:
		return corev1pb.MachineState_MachineState_Running
	case types.MachineStateDegraded:
		return corev1pb.MachineState_MachineState_Degraded
	case types.MachineStateError:
		return corev1pb.MachineState_MachineState_Error
	case types.MachineStateTerminating:
		return corev1pb.MachineState_MachineState_Terminating
	case types.MachineStateTerminated:
		return corev1pb.MachineState_MachineState_Terminated
	default:
		return corev1pb.MachineState_MachineState_Unknown
	}
}

func ProtoToMachineState(protoState corev1pb.MachineState) types.MachineState {
	switch protoState {
	case corev1pb.MachineState_MachineState_Pending:
		return types.MachineStatePending
	case corev1pb.MachineState_MachineState_Starting:
		return types.MachineStateStarting
	case corev1pb.MachineState_MachineState_Running:
		return types.MachineStateRunning
	case corev1pb.MachineState_MachineState_Degraded:
		return types.MachineStateDegraded
	case corev1pb.MachineState_MachineState_Error:
		return types.MachineStateError
	case corev1pb.MachineState_MachineState_Terminating:
		return types.MachineStateTerminating
	case corev1pb.MachineState_MachineState_Terminated:
		return types.MachineStateTerminated
	default:
		return ""
	}
}

func MachineDesiredStateToProto(state types.MachineDesiredState) corev1pb.MachineDesiredState {
	switch state {
	case types.MachineDesiredStatePending:
		return corev1pb.MachineDesiredState_MachineDesiredState_Pending
	case types.MachineDesiredStateRunning:
		return corev1pb.MachineDesiredState_MachineDesiredState_Running
	case types.MachineDesiredStateTerminated:
		return corev1pb.MachineDesiredState_MachineDesiredState_Terminated
	default:
		return corev1pb.MachineDesiredState_MachineDesiredState_Unknown
	}
}

func ProtoToMachineDesiredState(state corev1pb.MachineDesiredState) types.MachineDesiredState {
	switch state {
	case corev1pb.MachineDesiredState_MachineDesiredState_Pending:
		return types.MachineDesiredStatePending
	case corev1pb.MachineDesiredState_MachineDesiredState_Running:
		return types.MachineDesiredStateRunning
	case corev1pb.MachineDesiredState_MachineDesiredState_Terminated:
		return types.MachineDesiredStateTerminated
	default:
		return ""
	}
}

func ProtoToMachineSpec(specPb *corev1pb.MachineSpec) types.MachineSpec {
	spec := types.MachineSpec{
		Cpus:     specPb.Cpus,
		MemoryMB: specPb.MemoryMb,
	}

	return spec
}

func ProtoToContainerSpec(specPb *corev1pb.ContainerSpec) types.ContainerSpec {
	spec := types.ContainerSpec{
		Image:   specPb.Image,
		Env:     specPb.Env,
		Command: specPb.Command,
	}
	if specPb.Name != nil {
		spec.Name = *specPb.Name
	}
	if specPb.WorkingDir != nil {
		spec.WorkingDir = *specPb.WorkingDir
	}
	if specPb.Healthcheck != nil {
		spec.Healthcheck = &types.ContainerHealthcheckSpec{
			InitialDelaySeconds: specPb.Healthcheck.InitialDelaySeconds,
			PeriodSeconds:       specPb.Healthcheck.PeriodSeconds,
		}
		switch healthcheckType := specPb.Healthcheck.Type.(type) {
		case *corev1pb.ContainerHealthcheckSpec_Http:
			spec.Healthcheck.Http = &types.ContainerHttpHealthcheckSpec{
				Method:  healthcheckType.Http.Method,
				Path:    healthcheckType.Http.Path,
				Port:    healthcheckType.Http.Port,
				Headers: healthcheckType.Http.Headers,
			}
		}
	}
	return spec
}
