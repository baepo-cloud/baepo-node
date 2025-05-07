package machinecontroller

import (
	"context"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
	"github.com/nrednav/cuid2"
	"github.com/sourcegraph/conc/pool"
)

func (c *Controller) PublishEvent(event *corev1pb.MachineEvent) {
	go func() {
		c.eventsChan <- event
	}()
}

func (c *Controller) SubscribeToEvents(handler func(context.Context, *corev1pb.MachineEvent)) func() {
	c.eventHandlersLock.Lock()
	defer c.eventHandlersLock.Unlock()
	subscriptionID := cuid2.Generate()
	c.eventHandlers[subscriptionID] = handler

	return func() {
		c.eventHandlersLock.Lock()
		defer c.eventHandlersLock.Unlock()
		delete(c.eventHandlers, subscriptionID)
	}
}

func (c *Controller) startEventDispatcher(ctx context.Context) {
	defer close(c.eventsChan)

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-c.eventsChan:
			c.dispatchEvent(ctx, event)
		}
	}
}

func (c *Controller) dispatchEvent(ctx context.Context, event *corev1pb.MachineEvent) {
	c.eventHandlersLock.RLock()
	defer c.eventHandlersLock.RUnlock()

	p := pool.New().WithContext(ctx)
	for _, handler := range c.eventHandlers {
		p.Go(func(ctx context.Context) error {
			handler(context.Background(), event)
			return nil
		})
	}
	_ = p.Wait()
}
