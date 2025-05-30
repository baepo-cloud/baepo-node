package types

type (
	ContainerSpec struct {
		Name        string
		Image       string
		Env         map[string]string
		Command     []string
		Healthcheck *ContainerHealthcheckSpec
		WorkingDir  string
	}

	ContainerHealthcheckSpec struct {
		InitialDelaySeconds int32
		PeriodSeconds       int32
		Http                *ContainerHttpHealthcheckSpec
	}

	ContainerHttpHealthcheckSpec struct {
		Method  string
		Path    string
		Port    int32
		Headers map[string]string
	}
)
