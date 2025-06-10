package types

type (
	Container struct {
		ID   string
		Spec ContainerSpec
	}

	ContainerSpec struct {
		Name        *string
		Image       string
		Env         map[string]string
		Command     []string
		User        *string
		Healthcheck *ContainerHealthcheckSpec
		WorkingDir  *string
		Restart     *ContainerRestartSpec
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

	ContainerRestartPolicy string

	ContainerRestartSpec struct {
		Policy     ContainerRestartPolicy
		MaxRetries int32
	}
)

const (
	ContainerRestartPolicyUnknown   ContainerRestartPolicy = "unknown"
	ContainerRestartPolicyNo        ContainerRestartPolicy = "no"
	ContainerRestartPolicyOnFailure ContainerRestartPolicy = "on_failure"
	ContainerRestartPolicyAlways    ContainerRestartPolicy = "always"
)
