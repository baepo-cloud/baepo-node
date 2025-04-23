package v1pbadapter

import (
	"github.com/baepo-cloud/baepo-node/internal/types"
	v1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
)

func MachineStateToProto(state types.MachineState) v1pb.MachineState {
	switch state {
	case types.MachineStatePending:
		return v1pb.MachineState_MachineState_Pending
	case types.MachineStateStarting:
		return v1pb.MachineState_MachineState_Starting
	case types.MachineStateRunning:
		return v1pb.MachineState_MachineState_Running
	case types.MachineStateDegraded:
		return v1pb.MachineState_MachineState_Degraded
	case types.MachineStateError:
		return v1pb.MachineState_MachineState_Error
	case types.MachineStateTerminating:
		return v1pb.MachineState_MachineState_Terminating
	case types.MachineStateTerminated:
		return v1pb.MachineState_MachineState_Terminated
	default:
		return v1pb.MachineState_MachineState_Unknown
	}
}

func MachineDesiredStateToProto(state types.MachineDesiredState) v1pb.MachineDesiredState {
	switch state {
	case types.MachineDesiredStatePending:
		return v1pb.MachineDesiredState_MachineDesiredState_Pending
	case types.MachineDesiredStateRunning:
		return v1pb.MachineDesiredState_MachineDesiredState_Running
	case types.MachineDesiredStateTerminated:
		return v1pb.MachineDesiredState_MachineDesiredState_Terminated
	default:
		return v1pb.MachineDesiredState_MachineDesiredState_Unknown
	}
}

func ProtoToMachineDesiredState(state v1pb.MachineDesiredState) types.MachineDesiredState {
	switch state {
	case v1pb.MachineDesiredState_MachineDesiredState_Pending:
		return types.MachineDesiredStatePending
	case v1pb.MachineDesiredState_MachineDesiredState_Running:
		return types.MachineDesiredStateRunning
	case v1pb.MachineDesiredState_MachineDesiredState_Terminated:
		return types.MachineDesiredStateTerminated
	default:
		return ""
	}
}
