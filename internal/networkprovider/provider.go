package networkprovider

import (
	"bytes"
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/vishvananda/netlink"
	"gorm.io/gorm"
	"net"
	"os/exec"
	"strings"
	"sync"
)

type Provider struct {
	db              *gorm.DB
	bridgeInterface string
	networkCIDR     *net.IPNet
	networkAddr     net.IP
	gatewayAddr     net.IP
	allocatedIPs    []string // ip address = array offset + base address, value = tap interface Name
	lock            sync.Mutex
}

var _ types.NetworkProvider = (*Provider)(nil)

func New(db *gorm.DB) (*Provider, error) {
	_, networkCIDR, err := net.ParseCIDR("192.168.100.0/24")
	if err != nil {
		return nil, fmt.Errorf("invalid network CIDR: %w", err)
	}

	p := &Provider{
		db:              db,
		bridgeInterface: "br0",
		networkCIDR:     networkCIDR,
		lock:            sync.Mutex{},
	}

	_, err = p.SetupBridge("eth0")
	if err != nil {
		return nil, fmt.Errorf("failed to setup bridge interface: %w", err)
	}

	ones, bits := p.networkCIDR.Mask.Size()
	maxAddresses := 1 << (bits - ones)
	p.allocatedIPs = make([]string, maxAddresses)
	p.allocatedIPs[p.calculateOffsetFromIP(p.networkAddr)] = "network" // claim network address
	p.allocatedIPs[p.calculateOffsetFromIP(p.gatewayAddr)] = "br0"     // claim gateway address

	links, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	for _, link := range links {
		if tapName := link.Attrs().Name; strings.HasPrefix(tapName, "tap") {
			hwAddr := link.Attrs().HardwareAddr
			if index := p.calculateIndexFromHwAddr(hwAddr); index != -1 {
				p.allocatedIPs[index] = tapName
			}
		}
	}
	return p, nil
}

func (p *Provider) runCmd(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}

	return nil
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
