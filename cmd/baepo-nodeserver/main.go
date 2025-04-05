package main

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/baepo-app/baepo-node/internal/apiserver"
	"github.com/baepo-app/baepo-node/internal/gatewayserver"
	"github.com/baepo-app/baepo-node/internal/networkprovider"
	"github.com/baepo-app/baepo-node/internal/nodeservice"
	"github.com/baepo-app/baepo-node/internal/runtimeprovider"
	"github.com/baepo-app/baepo-node/internal/types"
	"github.com/baepo-app/baepo-node/internal/volumeprovider"
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
		fx.Provide(apiserver.New),
		fx.Provide(gatewayserver.New),
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
		fx.Invoke(func(lc fx.Lifecycle, server *apiserver.Server) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					return server.Start(ctx)
				},
				OnStop: func(ctx context.Context) error {
					return server.Stop(ctx)
				},
			})
		}),
		fx.Invoke(func(lc fx.Lifecycle, server *gatewayserver.Server) {
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

func provideConfig() (*types.NodeServerConfig, error) {
	config := types.NodeServerConfig{
		IPAddr:           os.Getenv("NODE_IP_ADDR"),
		ClusterID:        os.Getenv("NODE_CLUSTER_ID"),
		BootstrapToken:   os.Getenv("NODE_BOOTSTRAP_TOKEN"),
		APIAddr:          os.Getenv("NODE_API_ADDR"),
		GatewayAddr:      os.Getenv("NODE_GATEWAY_ADDR"),
		StorageDirectory: os.Getenv("NODE_STORAGE_DIRECTORY"),
	}
	if config.APIAddr == "" {
		config.APIAddr = ":3443"
	}
	if config.GatewayAddr == "" {
		config.GatewayAddr = ":8443"
	}
	if config.StorageDirectory == "" {
		config.StorageDirectory = "./storage"
	}
	if config.ClusterID == "" {
		return nil, errors.New("NODE_CLUSTER_ID env variable required")
	}
	if config.BootstrapToken == "" {
		return nil, errors.New("NODE_BOOTSTRAP_TOKEN env variable required")
	}
	return &config, nil
}

func provideVolumeProvider() types.VolumeProvider {
	return volumeprovider.New("vg_sandbox", "code_interpreter")
}

func provideRuntimeProvider(config *types.NodeServerConfig) types.RuntimeProvider {
	return runtimeprovider.New(
		"./resources/cloud-hypervisor",
		config.StorageDirectory,
		"./resources/vmlinux",
		"./resources/initramfs.cpio.gz",
	)
}

func provideApiClient() v1connect.NodeServiceClient {
	client := &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
			ReadIdleTimeout: 0,
			PingTimeout:     60 * time.Second,
		},
	}

	return v1connect.NewNodeServiceClient(client, "http://localhost:3000")
}
