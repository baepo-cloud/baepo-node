package types

type (
	ContainerState struct {
	}

	ContainerService interface {
		ContainersState() []*ContainerState
	}
)
