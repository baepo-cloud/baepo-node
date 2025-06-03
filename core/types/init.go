package types

type (
	InitConfig struct {
		IPAddress      string
		MacAddress     string
		GatewayAddress string
		Hostname       string
		Containers     []InitContainerConfig
	}

	InitContainerConfig struct {
		ContainerID   string
		ContainerName string
		Env           map[string]string
		Command       []string
		User          string
		WorkingDir    string
		Volume        string
		Healthcheck   *InitContainerHealthcheckConfig
		Restart       *InitContainerRestartConfig
	}

	InitContainerHealthcheckConfig struct {
		InitialDelaySeconds int32
		PeriodSeconds       int32
		Http                *InitContainerHttpHealthcheckConfig
	}

	InitContainerHttpHealthcheckConfig struct {
		Method  string
		Path    string
		Port    int32
		Headers map[string]string
	}

	InitContainerRestartConfig struct {
		Policy     RestartPolicy
		MaxRetries int32
	}

	RestartPolicy string
)

const (
	RestartPolicyNo        RestartPolicy = "no"
	RestartPolicyOnFailure RestartPolicy = "on_failure"
	RestartPolicyAlways    RestartPolicy = "always"

	InitServerPort = 9000
)
