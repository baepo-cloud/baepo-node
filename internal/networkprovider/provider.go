package networkprovider

import (
	"fmt"
	"github.com/baepo-app/baepo-node/internal/types"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"net"
	"strings"
	"sync"
)

type Provider struct {
	bridgeInterface string
	networkCIDR     *net.IPNet
	networkAddr     net.IP
	gatewayAddr     net.IP
	allocatedIPs    []string // ip address = array offset + base address, value = tap interface Name
	lock            sync.Mutex
}

var _ types.NetworkProvider = (*Provider)(nil)

func New() (*Provider, error) {
	bridgeInterface := "br0"
	bridge, err := netlink.LinkByName(bridgeInterface)
	if err != nil {
		return nil, fmt.Errorf("bridge interface br0 not found (maybe run ./scripts/setup-network?): %w", err)
	}

	brigeAddrs, err := netlink.AddrList(bridge, unix.AF_INET)
	if err != nil {
		return nil, fmt.Errorf("failed to list address of bridge interface: %w", err)
	}

	allocator := &Provider{
		bridgeInterface: bridgeInterface,
		networkCIDR:     brigeAddrs[0].IPNet,
		networkAddr:     brigeAddrs[0].IPNet.IP,
		gatewayAddr:     brigeAddrs[0].IP,
		lock:            sync.Mutex{},
	}

	ones, bits := allocator.networkCIDR.Mask.Size()
	maxAddresses := 1 << (bits - ones)
	allocator.allocatedIPs = make([]string, maxAddresses)
	allocator.allocatedIPs[0] = "network"                                                  // claim network address
	allocator.allocatedIPs[allocator.calculateOffsetFromIP(allocator.gatewayAddr)] = "br0" // claim gateway address

	links, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	for _, link := range links {
		if tapName := link.Attrs().Name; strings.HasPrefix(tapName, "tap") {
			hwAddr := link.Attrs().HardwareAddr
			if index := allocator.calculateIndexFromHwAddr(hwAddr); index != -1 {
				allocator.allocatedIPs[index] = tapName
			}
		}
	}
	return allocator, nil
}

func (p *Provider) calculateOffsetFromIP(ip net.IP) int {
	ipCopy := make(net.IP, len(ip))
	copy(ipCopy, ip)

	baseIPCopy := make(net.IP, len(p.networkAddr))
	copy(baseIPCopy, p.networkAddr)

	if ipCopy.To4() != nil {
		ipCopy = ipCopy.To4()
	}

	if baseIPCopy.To4() != nil {
		baseIPCopy = baseIPCopy.To4()
	}

	offset := 0
	for i := 0; i < len(ipCopy); i++ {
		offset = (offset << 8) | int(ipCopy[i]-baseIPCopy[i])
	}
	return offset
}

func (p *Provider) calculateIPFromOffset(offset int) net.IP {
	ip := make(net.IP, len(p.networkAddr))
	copy(ip, p.networkAddr)

	if ip.To4() != nil {
		ip = ip.To4()
	}

	for i := len(ip) - 1; i >= 0; i-- {
		ip[i] += byte(offset & 0xFF)
		offset >>= 8

		if offset == 0 {
			break
		}
	}
	return ip
}

func (p *Provider) calculateIndexFromHwAddr(hwAddr net.HardwareAddr) int {
	if len(hwAddr) == 6 && hwAddr[0] == 0x52 && hwAddr[1] == 0x54 && hwAddr[2] == 0x00 {
		index := (int(hwAddr[3]) << 16) | (int(hwAddr[4]) << 8) | int(hwAddr[5])
		if index >= 0 && index < len(p.allocatedIPs) {
			return index
		}
	}

	return -1
}
