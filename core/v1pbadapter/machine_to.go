package v1pbadapter

import (
	"github.com/baepo-cloud/baepo-node/core/types"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
)

func ToMachineSpec(spec *corev1pb.MachineSpec) *types.MachineSpec {
	return &types.MachineSpec{
		Name:     spec.Name,
		Cpus:     spec.Cpus,
		MemoryMB: spec.MemoryMb,
		Timeout:  spec.Timeout,
	}
}

func ToMachineDesiredState(state corev1pb.MachineDesiredState) types.MachineDesiredState {
	switch state {
	case corev1pb.MachineDesiredState_MachineDesiredState_Pending:
		return types.MachineDesiredStatePending
	case corev1pb.MachineDesiredState_MachineDesiredState_Running:
		return types.MachineDesiredStateRunning
	case corev1pb.MachineDesiredState_MachineDesiredState_Terminated:
		return types.MachineDesiredStateTerminated
	default:
		return types.MachineDesiredStateUnknown
	}
}

func ToMachineState(protoState corev1pb.MachineState) types.MachineState {
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
		return types.MachineStateUnknown
	}
}
