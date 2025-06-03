package eventbus

import (
	"context"
	"github.com/nrednav/cuid2"
	"github.com/sourcegraph/conc/pool"
	"sync"
)

type Bus[T any] struct {
	eventsChan        chan T
	eventHandlers     map[string]func(context.Context, T)
	eventHandlersLock sync.RWMutex
}

func NewBus[T any]() *Bus[T] {
	return &Bus[T]{
		eventsChan:        make(chan T),
		eventHandlers:     make(map[string]func(context.Context, T)),
		eventHandlersLock: sync.RWMutex{},
	}
}

func (b *Bus[T]) PublishEvent(event T) {
	go func() {
		b.eventsChan <- event
	}()
}

func (b *Bus[T]) SubscribeToEvents(handler func(context.Context, T)) func() {
	b.eventHandlersLock.Lock()
	defer b.eventHandlersLock.Unlock()
	subscriptionID := cuid2.Generate()
	b.eventHandlers[subscriptionID] = handler

	return func() {
		b.eventHandlersLock.Lock()
		defer b.eventHandlersLock.Unlock()
		delete(b.eventHandlers, subscriptionID)
	}
}

func (b *Bus[T]) StartDispatcher(ctx context.Context) {
	defer close(b.eventsChan)

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-b.eventsChan:
			b.dispatchEvent(ctx, event)
		}
	}
}

func (b *Bus[T]) dispatchEvent(ctx context.Context, event T) {
	b.eventHandlersLock.RLock()
	defer b.eventHandlersLock.RUnlock()

	p := pool.New().WithContext(ctx)
	for _, handler := range b.eventHandlers {
		p.Go(func(ctx context.Context) error {
			handler(ctx, event)
			return nil
		})
	}
	_ = p.Wait()
}
