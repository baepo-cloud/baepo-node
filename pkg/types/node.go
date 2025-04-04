package types

import (
	"context"
	"errors"
	"net"
)

type (
	Node struct {
		ID              string            `json:"id"`
		PublicIPAddress string            `json:"publicIpAddress"`
		ServerEndpoint  string            `json:"serverEndpoint"`
		VCpus           int               `json:"vCpus"`
		Memory          int64             `json:"memory"`
		Metadata        map[string]string `json:"metadata"`
	}

	NodeMachine struct {
		MachineID        string
		HypervisorPID    int
		Spec             *NodeMachineSpec
		Volume           *NodeVolume
		NetworkInterface *NodeNetworkInterface
	}

	NodeMachineSpec struct {
		Vcpus  int
		Memory int64
		Env    map[string]string
	}

	NodeVolume struct {
		ID       string
		Path     string
		ReadOnly bool
	}

	NodeNetworkInterface struct {
		Name       string
		IPAddress  net.IP
		MacAddress net.HardwareAddr
	}

	NodeStartMachineOptions struct {
		MachineID string
		Spec      NodeMachineSpec
	}

	NodeRegisterNodeOptions struct {
		PublicIPAddress string
		ServerEndpoint  string
		VCpus           int
		Memory          int64
		Metadata        map[string]string
	}

	NodeRegistryService interface {
		List(ctx context.Context) ([]*Node, error)

		Register(ctx context.Context, opts NodeRegisterNodeOptions) (*Node, error)
	}
)

var (
	ErrNoNodeAvailable     = errors.New("no node available")
	ErrNodeMachineNotFound = errors.New("node machine not found")
)
