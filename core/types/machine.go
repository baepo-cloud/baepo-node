package types

type (
	MachineState string

	MachineDesiredState string

	Machine struct {
		ID   string
		Spec MachineSpec
	}

	MachineSpec struct {
		Name     *string
		Cpus     uint32
		MemoryMB uint64
		Timeout  *uint64
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
	MachineStateUnknown     MachineState = ""

	MachineDesiredStatePending    MachineDesiredState = "pending"
	MachineDesiredStateRunning    MachineDesiredState = "running"
	MachineDesiredStateTerminated MachineDesiredState = "terminated"
	MachineDesiredStateUnknown    MachineDesiredState = ""
)
