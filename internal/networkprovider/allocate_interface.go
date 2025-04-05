package networkprovider

import (
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/vishvananda/netlink"
	"log/slog"
	"net"
	"os/exec"
	"strings"
)

func (p *Provider) AllocateInterface() (*types.NetworkInterface, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for index := 0; index < len(p.allocatedIPs); index++ {
		if p.allocatedIPs[index] != "" {
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
		ipAddress := p.calculateIPFromOffset(index)
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

		bridge, err := netlink.LinkByName(p.bridgeInterface)
		if err != nil {
			_ = netlink.LinkDel(tap)
			return nil, fmt.Errorf("failed to find bridge %s: %w", p.bridgeInterface, err)
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

		p.allocatedIPs[index] = tap.LinkAttrs.Name
		return &types.NetworkInterface{
			Name:       tapName,
			IPAddress:  ipAddress,
			MacAddress: hwAddress,
		}, nil
	}

	return nil, fmt.Errorf("no available ip addresses in network %s", p.networkCIDR)
}
