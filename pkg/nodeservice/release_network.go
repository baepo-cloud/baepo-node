package nodeservice

import (
	"context"
)

func (s *Service) ReleaseNetwork(ctx context.Context, interfaceName string) error {
	return s.networkAllocator.ReleaseInterface(interfaceName)
}
