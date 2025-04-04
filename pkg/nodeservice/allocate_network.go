package nodeservice

import "github.com/baepo-app/baepo-node/pkg/types"

func (s *Service) AllocateMachineNetwork() (*types.NodeNetworkInterface, error) {
	netInterface, err := s.networkAllocator.AllocateInterface()
	if err != nil {
		return nil, err
	}

	return &types.NodeNetworkInterface{
		Name:       netInterface.Name,
		IPAddress:  netInterface.IPAddress,
		MacAddress: netInterface.MacAddress,
	}, nil
}
