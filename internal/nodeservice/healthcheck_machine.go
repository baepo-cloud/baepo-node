package nodeservice

import (
	"context"
	"github.com/baepo-app/baepo-node/internal/types"
	"log/slog"
)

func (s *Service) HealthcheckMachine(ctx context.Context, machineID string) (*types.Machine, error) {
	machine, err := s.FindMachine(ctx, machineID)
	if err != nil {
		return nil, err
	}

	slog.Info("performing machine healthcheck", slog.String("machine-id", machineID))

	//httpClient := &http.Client{
	//	Transport: &http.Transport{
	//		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
	//			return vsock.Dial(s.getInitDaemonSocketPath(machineID), initd.ServerPort)
	//		},
	//	},
	//}
	//initClient := v1connect.NewInitDClient(httpClient, "http://initd")
	//_, err = initClient.Healthcheck(ctx, connect.NewRequest(&emptypb.Empty{}))

	//todo set machine state
	return machine, err
}
