package main

import (
	"context"
	"crypto/tls"
	"github.com/baepo-app/baepo-node/pkg/networkprovider"
	"github.com/baepo-app/baepo-node/pkg/nodeserver"
	"github.com/baepo-app/baepo-node/pkg/nodeservice"
	"github.com/baepo-app/baepo-node/pkg/runtimeprovider"
	"github.com/baepo-app/baepo-node/pkg/types"
	"github.com/baepo-app/baepo-node/pkg/volumeprovider"
	"github.com/baepo-app/baepo-oss/pkg/fxlog"
	"github.com/baepo-app/baepo-oss/pkg/proto/baepo/api/v1/v1connect"
	_ "github.com/joho/godotenv/autoload"
	"go.uber.org/fx"
	"golang.org/x/net/http2"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	fx.New(
		fxlog.Logger(),
		fx.Provide(provideConfig),
		fx.Provide(fx.Annotate(networkprovider.New, fx.As(new(types.NetworkProvider)))),
		fx.Provide(provideVolumeProvider),
		fx.Provide(provideRuntimeProvider),
		fx.Provide(provideApiClient),
		fx.Provide(fx.Annotate(nodeservice.New, fx.As(new(types.NodeService)))),
		fx.Provide(nodeserver.New),
		fx.Invoke(func(lc fx.Lifecycle, service types.NodeService) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					return service.Start(ctx)
				},
				OnStop: func(ctx context.Context) error {
					return service.Stop(ctx)
				},
			})
		}),
		fx.Invoke(func(lc fx.Lifecycle, server *nodeserver.Server) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					return server.Start(ctx)
				},
				OnStop: func(ctx context.Context) error {
					return server.Stop(ctx)
				},
			})
		}),
	).Run()
}

func provideConfig() types.NodeServerConfig {
	config := types.NodeServerConfig{
		IPAddr:           "127.0.0.1",
		ServerAddr:       os.Getenv("NODE_SERVER_ADDR"),
		GatewayAddr:      os.Getenv("NODE_GATEWAY_ADDR"),
		StorageDirectory: os.Getenv("NODE_STORAGE_DIRECTORY"),
	}
	if config.ServerAddr == "" {
		config.ServerAddr = ":3443"
	}
	if config.GatewayAddr == "" {
		config.GatewayAddr = ":8443"
	}
	if config.StorageDirectory == "" {
		config.StorageDirectory = "./storage"
	}
	return config
}

func provideVolumeProvider() types.VolumeProvider {
	return volumeprovider.New("vg_sandbox", "code_interpreter")
}

func provideRuntimeProvider(config types.NodeServerConfig) types.RuntimeProvider {
	return runtimeprovider.New(
		"./resources/cloud-hypervisor",
		config.StorageDirectory,
		"./resources/vmlinux",
		"./resources/initramfs.cpio.gz",
	)
}

func provideApiClient() v1connect.NodeServiceClient {
	transport := &http2.Transport{
		AllowHTTP: true, // For localhost dev
		DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		},
		ReadIdleTimeout: 0, // No timeout for streaming connections
		PingTimeout:     60 * time.Second,
	}

	client := &http.Client{Transport: transport}

	return v1connect.NewNodeServiceClient(client, "http://localhost:3000")
}
