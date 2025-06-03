package registrationservice

import (
	"context"
	"fmt"
	apiv1pb "github.com/baepo-cloud/baepo-proto/go/baepo/api/v1"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

func (c *Connection) sendStatsEvent(ctx context.Context) error {
	stats, err := c.newStatsProto(ctx)
	if err != nil {
		return fmt.Errorf("failed to create new stats event: %w", err)
	}

	return c.stream.Send(&apiv1pb.NodeControllerClientEvent{
		Event: &apiv1pb.NodeControllerClientEvent_Stats_{
			Stats: stats,
		},
	})
}

func (c *Connection) newStatsProto(ctx context.Context) (*apiv1pb.NodeControllerClientEvent_Stats, error) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, err
	}

	machines, err := c.service.machineService.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list machines: %w", err)
	}

	reservedMemoryMB := uint64(0)
	for _, machine := range machines {
		reservedMemoryMB += machine.Spec.MemoryMB
	}

	return &apiv1pb.NodeControllerClientEvent_Stats{
		TotalMemoryMb:    memInfo.Total / 1024 / 1024,
		UsedMemoryMb:     memInfo.Used / 1024 / 1024,
		ReservedMemoryMb: reservedMemoryMB,
		CpuCount:         uint32(len(cpuInfo)),
	}, nil
}
