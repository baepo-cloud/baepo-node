package nodeservice

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"log/slog"
	"net"
	"os/exec"
	"strings"
	"sync"
)

type (
	networkAllocator struct {
		bridgeInterface string
		networkCIDR     *net.IPNet
		networkAddr     net.IP
		gatewayAddr     net.IP
		allocatedIPs    []string // ip address = array offset + base address, value = tap interface Name
		lock            sync.Mutex
	}

	networkInterface struct {
		Name       string
		IPAddress  net.IP
		MacAddress net.HardwareAddr
	}
)

func newNetworkAllocator() (*networkAllocator, error) {
	bridgeInterface := "br0"
	bridge, err := netlink.LinkByName(bridgeInterface)
	if err != nil {
		return nil, fmt.Errorf("bridge interface br0 not found (maybe run ./scripts/setup-network?): %w", err)
	}

	brigeAddrs, err := netlink.AddrList(bridge, unix.AF_INET)
	if err != nil {
		return nil, fmt.Errorf("failed to list address of bridge interface: %w", err)
	}

	allocator := &networkAllocator{
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
			if len(hwAddr) == 6 && hwAddr[0] == 0x52 && hwAddr[1] == 0x54 && hwAddr[2] == 0x00 {
				index := (int(hwAddr[3]) << 16) | (int(hwAddr[4]) << 8) | int(hwAddr[5])
				if index >= 0 && index < len(allocator.allocatedIPs) {
					allocator.allocatedIPs[index] = tapName
				}
			}
		}
	}
	return allocator, nil
}

func (a *networkAllocator) AllocateInterface() (*networkInterface, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	for index := 0; index < len(a.allocatedIPs); index++ {
		if a.allocatedIPs[index] != "" {
			continue
		}

		tapName := fmt.Sprintf("tap%d", index)
		tap := &netlink.Tuntap{
			LinkAttrs: netlink.LinkAttrs{
				Name: tapName,
			},
			Mode:  netlink.TUNTAP_MODE_TAP,
			Flags: netlink.TUNTAP_ONE_QUEUE | netlink.TUNTAP_VNET_HDR,
		}
		ipAddress := a.calculateIPFromOffset(index)
		macAddress := fmt.Sprintf("52:54:00:%02x:%02x:%02x",
			(index>>16)&0xFF,
			(index>>8)&0xFF,
			index&0xFF)
		hwAddress, err := net.ParseMAC(macAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to parse mac address: %w", err)
		}

		if err = netlink.LinkAdd(tap); err != nil {
			return nil, fmt.Errorf("failed to add tap interface: %w", err)
		}

		if err = netlink.LinkSetHardwareAddr(tap, hwAddress); err != nil {
			_ = netlink.LinkDel(tap)
			return nil, fmt.Errorf("failed to set mac address: %w", err)
		}

		if err = netlink.LinkSetUp(tap); err != nil {
			_ = netlink.LinkDel(tap)
			return nil, fmt.Errorf("failed to set tap interface up: %w", err)
		}

		bridge, err := netlink.LinkByName(a.bridgeInterface)
		if err != nil {
			_ = netlink.LinkDel(tap)
			return nil, fmt.Errorf("failed to find bridge %s: %w", a.bridgeInterface, err)
		}

		if err = netlink.LinkSetMaster(tap, bridge); err != nil {
			_ = netlink.LinkDel(tap)
			return nil, fmt.Errorf("failed to set tap interface master: %w", err)
		}

		cmd := exec.Command("ebtables", "-A", "FORWARD", "-i", tapName, "-s", "!", strings.ToLower(macAddress), "-j", "DROP")
		if err = cmd.Run(); err != nil {
			slog.Error("failed to add mac filtering rule", slog.Any("error", err))
		}
		cmd = exec.Command("iptables", "-A", "FORWARD", "-i", tapName, "!", "-s", ipAddress.String(), "-j", "DROP")
		if err = cmd.Run(); err != nil {
			slog.Error("failed to add ip filtering rule", slog.Any("error", err))
		}
		cmd = exec.Command("arptables", "-A", "FORWARD", "-i", tapName, "--source-mac", "!", strings.ToLower(macAddress), "-j", "DROP")
		if err = cmd.Run(); err != nil {
			slog.Error("failed to add arp filtering rule", slog.Any("error", err))
		}

		a.allocatedIPs[index] = tap.LinkAttrs.Name
		return &networkInterface{
			Name:       tapName,
			IPAddress:  ipAddress,
			MacAddress: hwAddress,
		}, nil
	}

	return nil, fmt.Errorf("no available ip addresses in network %s", a.networkCIDR)
}

func (a *networkAllocator) ReleaseInterface(name string) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	index := -1
	for i, tapName := range a.allocatedIPs {
		if tapName == name {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("interface %s not found", name)
	}

	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("failed to find interface %s: %w", name, err)
	}

	if err = netlink.LinkDel(link); err != nil {
		return fmt.Errorf("failed to delete interface %s: %w", name, err)
	}

	if err = exec.Command("ebtables", "-D", "FORWARD", "-i", name, "-j", "DROP").Run(); err != nil {
		slog.Error("failed to remove mac filtering rule", slog.Any("error", err))
	}
	if err = exec.Command("iptables", "-D", "FORWARD", "-i", name, "-j", "DROP").Run(); err != nil {
		slog.Error("failed to remove ip filtering rule", slog.Any("error", err))
	}
	if err = exec.Command("arptables", "-D", "FORWARD", "-i", name, "-j", "DROP").Run(); err != nil {
		slog.Error("failed to remove arp filtering rule", slog.Any("error", err))
	}

	a.allocatedIPs[index] = ""
	return nil
}

func (a *networkAllocator) calculateOffsetFromIP(ip net.IP) int {
	ipCopy := make(net.IP, len(ip))
	copy(ipCopy, ip)

	baseIPCopy := make(net.IP, len(a.networkAddr))
	copy(baseIPCopy, a.networkAddr)

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

func (a *networkAllocator) calculateIPFromOffset(offset int) net.IP {
	ip := make(net.IP, len(a.networkAddr))
	copy(ip, a.networkAddr)

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
