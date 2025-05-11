package nodeservice

import (
	"context"
	"log/slog"
	"time"
)

func (s *Service) startGCWorker() {
	ticker := time.NewTicker(time.Hour)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.performGC()
		}
	}
}

func (s *Service) performGC() {
	startedAt := time.Now()
	s.log.Info("performing garbage collection")

	err := s.runtimeProvider.GC(context.Background(), func() []string {
		s.machineControllerLock.RLock()
		defer s.machineControllerLock.RUnlock()

		var machineIDs []string
		for id := range s.machineControllers {
			machineIDs = append(machineIDs, id)
		}
		return machineIDs
	})
	if err != nil {
		s.log.Error("failed to perform runtime garbage collection", slog.Any("error", err))
	}

	s.log.Info("garbage collection completed", slog.Duration("duration", time.Now().Sub(startedAt)))
}
