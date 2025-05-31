package containerservice

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/eventbus"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"github.com/baepo-cloud/baepo-node/init/internal/types"
	"github.com/nrednav/cuid2"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"sync/atomic"
	"time"
)

type Container struct {
	log              *slog.Logger
	config           coretypes.InitContainerConfig
	eventBus         *eventbus.Bus[any]
	stdout           io.Writer
	stderr           io.Writer
	cmd              *exec.Cmd
	startedAt        atomic.Pointer[time.Time]
	exitError        atomic.Pointer[error]
	exitedAt         atomic.Pointer[time.Time]
	restartCount     atomic.Int32
	healthy          atomic.Bool
	healthcheckError atomic.Pointer[error]
}

func (s *Service) StartContainer(config coretypes.InitContainerConfig) error {
	log := slog.With(slog.String("component", "container"), slog.String("container-id", config.ContainerID))
	log.Info("starting container")

	jsonConfig, err := json.Marshal(config)
	if err != nil {
		return err
	}

	stdout, stderr := s.logService.NewContainerLogWriter(config)
	ctr := &Container{
		log:      log,
		config:   config,
		eventBus: s.eventBus,
		stdout:   stdout,
		stderr:   stderr,
	}
	go ctr.run(jsonConfig)

	s.containersMutex.Lock()
	s.containers[config.ContainerID] = ctr
	s.containersMutex.Unlock()

	return nil
}

func (c *Container) run(jsonConfig []byte) {
	for {
		c.exitError.Store(nil)
		c.startedAt.Store(typeutil.Ptr(time.Now()))
		c.exitedAt.Store(nil)

		c.cmd = exec.Command("/initcontainer", string(jsonConfig))
		c.cmd.Stdout = c.stdout
		c.cmd.Stderr = c.stderr
		c.cmd.Stdin = os.Stdin
		if err := c.cmd.Start(); err != nil {
			c.log.Warn("failed to start container", slog.Any("error", err))
			c.exitError.Store(&err)
		} else {
			healthcheckCtx, cancel := context.WithCancel(context.Background())
			go c.healthcheckWorker(healthcheckCtx)
			err = c.cmd.Wait()
			c.log.Warn("container exited",
				slog.Any("error", err),
				slog.Int("exit-code", c.cmd.ProcessState.ExitCode()))
			c.exitError.Store(&err)
			c.exitedAt.Store(typeutil.Ptr(time.Now()))
			cancel()
		}
		c.eventBus.PublishEvent(c.newContainerStateChangedEvent())

		if c.config.Restart == nil || c.config.Restart.Policy == coretypes.RestartPolicyNo {
			return
		}

		exitCode := c.cmd.ProcessState.ExitCode()
		if c.config.Restart.Policy == coretypes.RestartPolicyOnFailure &&
			(exitCode == 0 || (c.config.Restart.MaxRetries > 0 && c.restartCount.Load() >= c.config.Restart.MaxRetries)) {
			return
		}

		c.restartCount.Add(1)
	}
}

func (c *Container) healthcheckWorker(ctx context.Context) {
	if c.config.Healthcheck == nil {
		c.healthy.Store(true)
		c.eventBus.PublishEvent(c.newContainerStateChangedEvent())
		return
	}

	timer := time.NewTimer(time.Duration(c.config.Healthcheck.InitialDelaySeconds) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			err := c.performHealthcheck(ctx)
			healthy := err == nil
			wasHealthy := c.healthy.Swap(healthy)
			c.healthcheckError.Store(&err)
			if wasHealthy != healthy {
				c.eventBus.PublishEvent(c.newContainerStateChangedEvent())
			}

			timer.Reset(time.Duration(c.config.Healthcheck.PeriodSeconds) * time.Second)
		}
	}
}

func (c *Container) performHealthcheck(ctx context.Context) error {
	spec := c.config.Healthcheck.Http
	if spec == nil {
		return nil
	}

	url := fmt.Sprintf("http://127.0.0.1:%d%s", spec.Port, spec.Path)
	req, err := http.NewRequestWithContext(ctx, spec.Method, url, nil)
	if err != nil {
		return err
	}
	for key, value := range spec.Headers {
		req.Header.Add(key, value)
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("invalid status code: %d", res.StatusCode)
	}
	return nil
}

func (c *Container) newContainerStateChangedEvent() *types.ContainerStateChangedEvent {
	event := &types.ContainerStateChangedEvent{
		EventID:      cuid2.Generate(),
		ContainerID:  c.config.ContainerID,
		Healthy:      c.healthy.Load(),
		RestartCount: c.restartCount.Load(),
		StartedAt:    c.startedAt.Load(),
		ExitedAt:     c.exitedAt.Load(),
		Timestamp:    time.Now(),
	}
	if err := c.healthcheckError.Load(); err != nil {
		event.HealthcheckError = *err
	}
	if err := c.exitError.Load(); err != nil {
		event.ExitError = *err
	}
	if c.cmd != nil && c.cmd.ProcessState != nil {
		if code := c.cmd.ProcessState.ExitCode(); code != -1 {
			event.ExitCode = typeutil.Ptr(int32(code))
		}
	}
	return event
}
