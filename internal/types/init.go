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
		Name        string
		Env         map[string]string
		Command     []string
		Healthcheck *MachineContainerHealthcheckSpec
		User        string
		WorkingDir  string
		Volume      string
	}

	InitContainerState struct {
	}

	InitService interface {
		ContainersState() []*InitContainerState
	}
)
