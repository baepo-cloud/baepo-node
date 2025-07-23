package networkprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"github.com/nrednav/cuid2"
	"net"
	"time"
)

func (p *Provider) AllocateInterface(ctx context.Context) (*types.NetworkInterface, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	index := p.findAvailableOffset()
	if index == -1 {
		return nil, fmt.Errorf("no available ip addresses in network %s", p.networkCIDR)
	}

	networkInterface := &types.NetworkInterface{
		ID:             cuid2.Generate(),
		Name:           fmt.Sprintf("tap%d", index),
		GatewayAddress: p.gatewayAddr,
		NetworkCIDR:    typeutil.Ptr(types.GormNetIPNet(*p.networkCIDR)),
		IPAddress:      p.calculateIPFromOffset(index),
		AllocatedAt:    typeutil.Ptr(time.Now()),
	}

	macAddress := fmt.Sprintf("52:54:00:%02x:%02x:%02x", (index>>16)&0xFF, (index>>8)&0xFF, index&0xFF)
	hwAddress, err := net.ParseMAC(macAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse mac address: %w", err)
	}
	networkInterface.MacAddress = hwAddress

	if err = p.db.WithContext(ctx).Create(&networkInterface).Error; err != nil {
		return nil, fmt.Errorf("failed to create network interface in database: %w", err)
	}

	p.allocatedIPs[index] = networkInterface.Name
	return networkInterface, nil
}

func (p *Provider) findAvailableOffset() int {
	for index := 0; index < len(p.allocatedIPs); index++ {
		if p.allocatedIPs[index] == "" {
			return index
		}
	}
	return -1
}
