package types

import "net"

type (
	NetworkInterface struct {
		Name           string
		IPAddress      net.IP
		MacAddress     net.HardwareAddr
		GatewayAddress net.IP
		NetworkCIDR    *net.IPNet
	}

	NetworkProvider interface {
		GetInterface(name string) (*NetworkInterface, error)

		AllocateInterface() (*NetworkInterface, error)

		ReleaseInterface(name string) error
	}
)
