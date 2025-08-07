package runtime

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/logmanager"
	"github.com/baepo-cloud/baepo-node/vmruntime/internal/chclient"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
	"github.com/baepo-cloud/baepo-proto/go/baepo/node/v1/nodev1pbconnect"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

type connectHandler struct {
	runtime *Runtime
}

var _ nodev1pbconnect.RuntimeHandler = (*connectHandler)(nil)

func (r *Runtime) startConnectServer() error {
	unixSocket := filepath.Join(r.config.WorkingDir, "runtime.sock")
	_ = os.Remove(unixSocket)

	ln, err := net.Listen("unix", unixSocket)
	if err != nil {
		return fmt.Errorf("failed to setup unix socket: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle(nodev1pbconnect.NewRuntimeHandler(&connectHandler{runtime: r}))
	r.httpServer = &http.Server{Handler: mux}
	go r.httpServer.Serve(ln)
	return nil
}

func (h *connectHandler) GetState(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[nodev1pb.RuntimeGetStateResponse], error) {
	res, _ := h.runtime.vmmClient.GetVmInfoWithResponse(ctx)

	return connect.NewResponse(&nodev1pb.RuntimeGetStateResponse{
		Pid:     int64(os.Getpid()),
		Running: res != nil && res.JSON200 != nil && res.JSON200.State == chclient.Running,
	}), nil
}

func (h *connectHandler) GetLogs(ctx context.Context, req *connect.Request[nodev1pb.RuntimeGetLogsRequest], stream *connect.ServerStream[nodev1pb.RuntimeGetLogsResponse]) error {
	initialLogs, err := h.runtime.logManager.ReadLogs(ctx)
	if err != nil {
		return err
	}

	for entry := range initialLogs {
		if entry.Source != logmanager.MachineLogEntrySource {
			continue
		}

		err = stream.Send(&nodev1pb.RuntimeGetLogsResponse{
			Timestamp: timestamppb.New(entry.Timestamp),
			Content:   []byte(entry.Message),
		})
		if err != nil {
			return err
		}
	}
	if !req.Msg.Follow {
		return nil
	}

	liveLogs := make(chan logmanager.Entry, 100)
	cancelHandler := h.runtime.logManager.HandleLogs(func(entry logmanager.Entry) {
		select {
		case liveLogs <- entry:
		case <-ctx.Done():
		}
	})
	defer cancelHandler()

	for {
		select {
		case <-ctx.Done():
			return nil
		case entry := <-liveLogs:
			if entry.Source != logmanager.MachineLogEntrySource {
				continue
			}

			err = stream.Send(&nodev1pb.RuntimeGetLogsResponse{
				Timestamp: timestamppb.New(entry.Timestamp),
				Content:   []byte(entry.Message),
			})
			if err != nil {
				return err
			}
		}
	}
}

func (h *connectHandler) GetContainerLogs(ctx context.Context, req *connect.Request[nodev1pb.RuntimeGetContainerLogsRequest], stream *connect.ServerStream[nodev1pb.RuntimeGetContainerLogsResponse]) error {
	//TODO implement me
	panic("implement me")
}

func (h *connectHandler) Events(ctx context.Context, req *connect.Request[emptypb.Empty], writeStream *connect.ServerStream[nodev1pb.RuntimeEventsResponse]) error {
	initClient, closeClient := h.runtime.newInitClient()
	defer closeClient()

	readStream, err := initClient.Events(ctx, connect.NewRequest(&emptypb.Empty{}))
	if err != nil {
		return err
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		} else if !readStream.Receive() {
			continue
		}

		if runtimeEvent := h.mapInitEvent(readStream.Msg()); runtimeEvent != nil {
			if err = writeStream.Send(runtimeEvent); err != nil {
				return err
			}
		}
	}
}

func (h *connectHandler) mapInitEvent(baseEvent *nodev1pb.InitEventsResponse) *nodev1pb.RuntimeEventsResponse {
	switch event := baseEvent.Event.(type) {
	case *nodev1pb.InitEventsResponse_ContainerStateChanged:
		return &nodev1pb.RuntimeEventsResponse{
			EventId:   baseEvent.EventId,
			Timestamp: baseEvent.Timestamp,
			Event: &nodev1pb.RuntimeEventsResponse_ContainerStateChanged{
				ContainerStateChanged: &nodev1pb.RuntimeEventsResponse_ContainerStateChangedEvent{
					ContainerId:      event.ContainerStateChanged.ContainerId,
					State:            event.ContainerStateChanged.State,
					StartedAt:        event.ContainerStateChanged.StartedAt,
					ExitedAt:         event.ContainerStateChanged.ExitedAt,
					ExitCode:         event.ContainerStateChanged.ExitCode,
					ExitError:        event.ContainerStateChanged.ExitError,
					Healthy:          event.ContainerStateChanged.Healthy,
					HealthcheckError: event.ContainerStateChanged.HealthcheckError,
					RestartCount:     event.ContainerStateChanged.RestartCount,
				},
			},
		}
	case *nodev1pb.InitEventsResponse_Ping:
		return &nodev1pb.RuntimeEventsResponse{
			EventId:   baseEvent.EventId,
			Timestamp: baseEvent.Timestamp,
			Event: &nodev1pb.RuntimeEventsResponse_Ping{
				Ping: &nodev1pb.RuntimeEventsResponse_PingEvent{},
			},
		}
	default:
		return nil
	}
}
