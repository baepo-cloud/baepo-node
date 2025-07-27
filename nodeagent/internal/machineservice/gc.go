package machineservice

import (
	"context"
	"log/slog"
	"time"
)

func (s *Service) startGCWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.performGC()
		}
	}
}

func (s *Service) performGC() {
	startedAt := time.Now()
	s.log.Info("performing garbage collection")

	//err := s.runtimeProvider.GC(context.Background(), func() []string {
	//	var machineIDs []string
	//	s.machineControllers.ForEach(func(machineID string, _ *machinecontroller.Controller) bool {
	//		machineIDs = append(machineIDs, machineID)
	//		return true
	//	})
	//	return machineIDs
	//})
	//if err != nil {
	//	s.log.Error("failed to perform runtime garbage collection", slog.Any("error", err))
	//}

	s.log.Info("garbage collection completed", slog.Duration("duration", time.Now().Sub(startedAt)))
}
