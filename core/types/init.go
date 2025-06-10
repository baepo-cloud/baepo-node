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
		ContainerSpec
		ContainerID string
		Volume      string
	}
)

const InitServerPort = 9000
