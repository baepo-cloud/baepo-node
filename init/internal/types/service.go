package types

type (
	InitContainerState struct {
	}

	InitService interface {
		ContainersState() []*InitContainerState
	}
)
