package types

import "net"

type (
	NetworkInterface struct {
		Name       string
		IPAddress  net.IP
		MacAddress net.HardwareAddr
	}

	NetworkProvider interface {
		GetInterface(name string) (*NetworkInterface, error)

		AllocateInterface() (*NetworkInterface, error)

		ReleaseInterface(name string) error
	}
)
