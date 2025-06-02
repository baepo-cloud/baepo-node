package registrationservice

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	apiv1pb "github.com/baepo-cloud/baepo-proto/go/baepo/api/v1"
	"os"
	"path"
)

func (c *Connection) sendRegistrationEvent(ctx context.Context) (*apiv1pb.NodeControllerServerEvent_RegistrationCompletedEvent, error) {
	nodeTokenFilePath := path.Join(c.service.config.StorageDirectory, "token")

	var nodeToken *string
	if b, err := os.ReadFile(nodeTokenFilePath); err == nil {
		nodeToken = typeutil.Ptr(string(b))
	}

	stats, err := c.newStatsProto(ctx)
	if err != nil {
		return nil, err
	}

	err = c.stream.Send(&apiv1pb.NodeControllerClientEvent{
		Event: &apiv1pb.NodeControllerClientEvent_Register_{
			Register: &apiv1pb.NodeControllerClientEvent_Register{
				ClusterId:       c.service.config.ClusterID,
				BootstrapToken:  c.service.config.BootstrapToken,
				NodeToken:       nodeToken,
				IpAddress:       c.service.config.IPAddr,
				ApiEndpoint:     c.service.getEndpoint(c.service.config.APIAddr),
				GatewayEndpoint: c.service.getEndpoint(c.service.config.GatewayAddr),
				Stats:           stats,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send registration request: %w", err)
	}

	event, err := c.stream.Receive()
	if err != nil {
		return nil, fmt.Errorf("failed to receive registration response: %w", err)
	}

	registrationCompleted, ok := event.Event.(*apiv1pb.NodeControllerServerEvent_RegistrationCompleted)
	if !ok {
		return nil, fmt.Errorf("received registration response is not valid: %v", event.Event)
	}

	authorityCert, err := parseCertificate(registrationCompleted.RegistrationCompleted.AuthorityCert)
	if err != nil {
		return nil, fmt.Errorf("failed to parse authority certificate: %w", err)
	}

	c.service.authorityCert = authorityCert
	tlsCert, err := tls.X509KeyPair(
		registrationCompleted.RegistrationCompleted.ServerCert,
		registrationCompleted.RegistrationCompleted.ServerKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load server tls certificate: %w", err)
	}
	c.service.tlsCert = &tlsCert

	err = os.WriteFile(nodeTokenFilePath, []byte(registrationCompleted.RegistrationCompleted.NodeToken), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to store node token: %w", err)
	}
	
	return registrationCompleted.RegistrationCompleted, nil
}
