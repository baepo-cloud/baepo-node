package machinecontroller

import (
	"context"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"log/slog"
	"reflect"
	"time"
)

func (c *Controller) eventHandler(ctx context.Context, anyEvent any) {
	c.log.Debug("handling event", slog.Any("event", anyEvent), slog.String("type", reflect.TypeOf(anyEvent).Elem().Name()))

	switch event := anyEvent.(type) {
	case *AssessStateMessage:
		c.log.Debug("checking if reconciliation is needed")
		state := c.GetState()
		if c.shouldReconcile(state.Machine) {
			c.reconcile()
		}
		shouldStartRuntimeListener := c.shouldStartRuntimeListener(state.Machine)
		if shouldStartRuntimeListener && state.RuntimeListener == nil {
			c.startRuntimeListener(state.Machine)
		} else if !shouldStartRuntimeListener && state.RuntimeListener != nil {
			state.RuntimeListener.Cancel()
		}
	case *DesiredStateChangedMessage:
		c.log.Debug("desired state changed", slog.String("new-state", string(event.DesiredState)))
		_ = c.SetState(func(s *State) error {
			s.Machine.DesiredState = event.DesiredState
			return c.db.WithContext(ctx).Select("DesiredState").Save(&s.Machine).Error
		})
		c.eventBus.PublishEvent(&AssessStateMessage{})
	case *StateChangedMessage:
		c.log.Debug("machine state changed", slog.String("new-state", string(event.State)))
		_ = c.SetState(func(s *State) error {
			s.Machine.State = event.State
			s.Machine.TerminatedAt = nil
			if event.State == coretypes.MachineStateTerminated {
				s.Machine.TerminatedAt = typeutil.Ptr(time.Now())
			}
			return c.db.WithContext(ctx).Select("State", "TerminatedAt").Save(&s.Machine).Error
		})
		c.eventBus.PublishEvent(&AssessStateMessage{})
	case *RuntimeListenerConnectedMessage:
		if state := c.GetState(); state.Machine.State != coretypes.MachineStateRunning {
			c.eventBus.PublishEvent(NewStateChangedMessage(coretypes.MachineStateRunning))
		}
		_ = c.SetState(func(s *State) error {
			s.RuntimeListener.ConsecutiveErrorCount = 0
			return nil
		})
	case *RuntimeListenerDisconnectedMessage:
		_ = c.SetState(func(s *State) error {
			if s.RuntimeListener != nil {
				s.RuntimeListener.ConsecutiveErrorCount++
			}
			return nil
		})
		state := c.GetState()
		if state.RuntimeListener == nil {
			return
		}

		if state.RuntimeListener.ConsecutiveErrorCount == 3 {
			c.eventBus.PublishEvent(NewStateChangedMessage(coretypes.MachineStateError))
		} else if state.RuntimeListener.ConsecutiveErrorCount == 1 {
			c.eventBus.PublishEvent(NewStateChangedMessage(coretypes.MachineStateDegraded))
		}
	}
}
