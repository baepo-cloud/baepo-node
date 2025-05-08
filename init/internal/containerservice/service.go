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
}

var _ types.ContainerService = (*Service)(nil)

func New(logService types.LogService) *Service {
	return &Service{
		logService: logService,
		containers: map[string]*Container{},
		eventBus:   eventbus.NewBus[any](),
	}
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
		defer close(events)
		defer cancel()
		<-ctx.Done()
	}()

	return events
}
