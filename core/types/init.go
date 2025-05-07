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
		Name       string
		Env        map[string]string
		Command    []string
		User       string
		WorkingDir string
		Volume     string
	}
)

const InitServerPort = 9000
