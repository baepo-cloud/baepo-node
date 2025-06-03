package registrationservice

import (
	"context"
	"log/slog"
	"time"
)

func (s *Service) startRegistrationWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			connCtx, cancelConn := context.WithCancel(ctx)
			err := s.openConnection(connCtx)
			cancelConn()
			if err != nil {
				s.log.Error("failed to register node, retrying in 5 seconds", slog.Any("error", err))
				select {
				case <-time.After(5 * time.Second):
					continue
				case <-ctx.Done():
					return
				}
			}
		}
	}
}
