package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/apiserver"
	"github.com/baepo-cloud/baepo-node/internal/fxlog"
	"github.com/baepo-cloud/baepo-node/internal/gatewayserver"
	"github.com/baepo-cloud/baepo-node/internal/networkprovider"
	"github.com/baepo-cloud/baepo-node/internal/nodeservice"
	"github.com/baepo-cloud/baepo-node/internal/runtimeprovider"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/baepo-cloud/baepo-node/internal/volumeprovider"
	"github.com/baepo-cloud/baepo-proto/go/baepo/api/v1/v1connect"
	_ "github.com/joho/godotenv/autoload"
	"go.uber.org/fx"
	"golang.org/x/net/http2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

func main() {
	//slog.SetLogLoggerLevel(slog.LevelDebug)
	fx.New(
		fxlog.Logger(),
		fx.Provide(provideConfig),
		fx.Provide(provideGORM),
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
		ClusterID:             os.Getenv("NODE_CLUSTER_ID"),
		BootstrapToken:        os.Getenv("NODE_BOOTSTRAP_TOKEN"),
		IPAddr:                os.Getenv("NODE_IP_ADDR"),
		APIAddr:               os.Getenv("NODE_API_ADDR"),
		GatewayAddr:           os.Getenv("NODE_GATEWAY_ADDR"),
		StorageDirectory:      os.Getenv("NODE_STORAGE_DIRECTORY"),
		InitBinary:            os.Getenv("NODE_INIT_BINARY"),
		VMLinux:               os.Getenv("NODE_VM_LINUX"),
		CloudHypervisorBinary: os.Getenv("NODE_CLOUD_HYPERVISOR_BINARY"),
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
	if config.InitBinary == "" {
		config.InitBinary = "./resources/baepo-initd"
	}
	if config.VMLinux == "" {
		config.VMLinux = "./resources/vmlinux"
	}
	if config.CloudHypervisorBinary == "" {
		config.CloudHypervisorBinary = "./resources/cloud-hypervisor"
	}
	if config.ClusterID == "" {
		return nil, errors.New("NODE_CLUSTER_ID env variable required")
	}
	if config.BootstrapToken == "" {
		return nil, errors.New("NODE_BOOTSTRAP_TOKEN env variable required")
	}
	if !filepath.IsAbs(config.InitBinary) {
		absPath, err := filepath.Abs(config.InitBinary)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path of init binary: %w", err)
		}
		config.InitBinary = absPath
	}
	if !filepath.IsAbs(config.StorageDirectory) {
		absPath, err := filepath.Abs(config.StorageDirectory)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path of the storage directory: %w", err)
		}
		config.StorageDirectory = absPath
	}

	return &config, nil
}

func provideGORM(config *types.NodeServerConfig) (*gorm.DB, error) {
	dbName := "node.db?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL"
	db, err := gorm.Open(sqlite.Open(path.Join(config.StorageDirectory, dbName)), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&types.Machine{}, &types.Volume{}, &types.NetworkInterface{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func provideVolumeProvider(db *gorm.DB) types.VolumeProvider {
	vg := os.Getenv("NODE_VOLUME_GROUP")
	if vg == "" {
		vg = "vg_baepo"
	}

	return volumeprovider.New(db, vg)
}

func provideRuntimeProvider(config *types.NodeServerConfig) types.RuntimeProvider {
	return runtimeprovider.New(
		config.CloudHypervisorBinary,
		config.InitBinary,
		config.StorageDirectory,
		config.VMLinux,
	)
}

func provideApiClient() v1connect.NodeControllerServiceClient {
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

	return v1connect.NewNodeControllerServiceClient(client, "http://localhost:3000")
}
