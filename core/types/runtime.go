package types

import "net"

type (
	RuntimeConfig struct {
		WorkingDir string
		MachineID  string
		Cpus       uint32
		MemoryMB   uint64
		Network    RuntimeNetworkConfig
		Containers []RuntimeContainerConfig
	}

	RuntimeNetworkConfig struct {
		InterfaceName  string
		IPAddress      net.IP
		MacAddress     net.HardwareAddr
		GatewayAddress net.IP
		NetworkCIDR    net.IPNet
		Hostname       string
	}

	RuntimeContainerConfig struct {
		ContainerSpec
		ContainerID string
		VolumePath  string
	}
)
