package networkprovider

import (
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/vishvananda/netlink"
)

func (p *Provider) GetInterface(name string) (*types.NetworkInterface, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, err
	}

	return &types.NetworkInterface{
		Name:       name,
		IPAddress:  p.calculateIPFromOffset(p.calculateIndexFromHwAddr(link.Attrs().HardwareAddr)),
		MacAddress: link.Attrs().HardwareAddr,
	}, nil
}
