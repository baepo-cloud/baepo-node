package containerservice

import (
	"context"
	"github.com/baepo-cloud/baepo-node/core/eventbus"
	"github.com/baepo-cloud/baepo-node/init/internal/types"
	"sync"
)

type Service struct {
	logService            types.LogService
	containersMutex       sync.RWMutex
	containers            map[string]*Container
	eventBus              *eventbus.Bus[any]
	cancelEventDispatcher context.CancelFunc
	previousEventsMutex   sync.RWMutex
	previousEvents        []any
}

var _ types.ContainerService = (*Service)(nil)

func New(logService types.LogService) *Service {
	srv := &Service{
		logService: logService,
		containers: map[string]*Container{},
		eventBus:   eventbus.NewBus[any](),
	}
	srv.eventBus.SubscribeToEvents(func(ctx context.Context, event any) {
		srv.previousEventsMutex.Lock()
		defer srv.previousEventsMutex.Unlock()
		srv.previousEvents = append(srv.previousEvents, event)
	})
	return srv
}

func (s *Service) Start() {
	dispatcherCtx, cancelDispatcher := context.WithCancel(context.Background())
	s.cancelEventDispatcher = cancelDispatcher
	go s.eventBus.StartDispatcher(dispatcherCtx)
}

func (s *Service) Stop() {
	if s.cancelEventDispatcher != nil {
		s.cancelEventDispatcher()
	}
}

func (s *Service) Events(ctx context.Context) <-chan any {
	events := make(chan any)
	cancel := s.eventBus.SubscribeToEvents(func(ctx context.Context, event any) {
		select {
		case <-ctx.Done():
			return
		case events <- event:
		}
	})

	go func() {
		s.previousEventsMutex.RLock()
		defer s.previousEventsMutex.RUnlock()

		for _, event := range s.previousEvents {
			events <- event
		}
	}()

	go func() {
		defer close(events)
		defer cancel()
		<-ctx.Done()
	}()

	return events
}
