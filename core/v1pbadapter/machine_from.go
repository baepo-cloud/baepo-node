package v1pbadapter

import (
	"github.com/baepo-cloud/baepo-node/core/types"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
)

func FromMachineState(state types.MachineState) corev1pb.MachineState {
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

func FromMachineDesiredState(state types.MachineDesiredState) corev1pb.MachineDesiredState {
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
