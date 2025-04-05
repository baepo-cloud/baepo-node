package main

import (
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/initd"
	"github.com/vishvananda/netlink"
	"net"
	"os"
)

func main() {
	ipAddr, err := netlink.ParseAddr(os.Args[1])
	if err != nil {
		panic(fmt.Errorf("failed to parse ip address: %w", err))
	}

	macAddr, err := net.ParseMAC(os.Args[2])
	if err != nil {
		panic(fmt.Errorf("failed to parse mac address: %w", err))
	}

	err = initd.Run(initd.Config{
		IPAddress:      ipAddr,
		MacAddress:     macAddr,
		GatewayAddress: net.ParseIP("192.168.100.1"),
	})
	if err != nil {
		panic(err)
	}
}
