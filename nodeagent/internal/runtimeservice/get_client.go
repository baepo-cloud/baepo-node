package runtimeservice

import (
	"context"
	"github.com/baepo-cloud/baepo-proto/go/baepo/node/v1/nodev1pbconnect"
	"net"
	"net/http"
	"path"
	"time"
)

func (s *Service) GetClient(machineID string) (nodev1pbconnect.RuntimeClient, func()) {
	var conns []net.Conn
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				conn, err := net.Dial("unix", path.Join(s.GetMachineDirectory(machineID), "runtime.sock"))
				if err != nil {
					return nil, err
				}

				conns = append(conns, conn)
				return conn, nil
			},
			IdleConnTimeout:       10 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
		},
	}
	return nodev1pbconnect.NewRuntimeClient(httpClient, "http://runtime"), func() {
		for _, conn := range conns {
			_ = conn.Close()
		}
	}
}
