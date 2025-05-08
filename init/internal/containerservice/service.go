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
	eventBus              *eventbus.Bus[types.ContainerEvent]
	cancelEventDispatcher context.CancelFunc
}

var _ types.ContainerService = (*Service)(nil)

func New(logService types.LogService) *Service {
	return &Service{
		logService: logService,
		containers: map[string]*Container{},
		eventBus:   eventbus.NewBus[types.ContainerEvent](),
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

func (s *Service) Events(ctx context.Context) <-chan types.ContainerEvent {
	events := make(chan types.ContainerEvent)
	cancel := s.eventBus.SubscribeToEvents(func(ctx context.Context, event types.ContainerEvent) {
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
