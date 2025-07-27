package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/fxlog"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/apiserver"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/gatewayserver"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/imageprovider"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/machineservice"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/networkprovider"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/registrationservice"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/runtimeservice"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/volumeprovider"
	"github.com/baepo-cloud/baepo-proto/go/baepo/api/v1/apiv1pbconnect"
	_ "github.com/joho/godotenv/autoload"
	"go.uber.org/fx"
	"golang.org/x/net/http2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	fx.New(
		fxlog.Logger(),
		fx.Provide(provideConfig),
		fx.Provide(provideGORM),
		fx.Provide(fx.Annotate(networkprovider.New, fx.As(new(types.NetworkProvider)))),
		fx.Provide(fx.Annotate(imageprovider.New, fx.As(new(types.ImageProvider)))),
		fx.Provide(fx.Annotate(runtimeservice.New, fx.As(new(types.RuntimeService)))),
		fx.Provide(fx.Annotate(volumeprovider.New, fx.As(new(types.VolumeProvider)))),
		fx.Provide(provideControlPlaneApiClient),
		fx.Provide(machineservice.New),
		fx.Provide(registrationservice.New),
		fx.Provide(apiserver.New),
		fx.Provide(gatewayserver.New),
		fx.Provide(func(lc fx.Lifecycle, service *machineservice.Service) types.MachineService {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					return service.Start(ctx)
				},
				OnStop: func(ctx context.Context) error {
					return service.Stop(ctx)
				},
			})
			return service
		}),
		fx.Provide(func(lc fx.Lifecycle, service *registrationservice.Service) types.RegistrationService {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					return service.Start(ctx)
				},
				OnStop: func(ctx context.Context) error {
					return service.Stop(ctx)
				},
			})
			return service
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

func provideConfig() (*types.Config, error) {
	config := types.Config{
		ClusterID:        os.Getenv("NODE_CLUSTER_ID"),
		BootstrapToken:   os.Getenv("NODE_BOOTSTRAP_TOKEN"),
		IPAddr:           os.Getenv("NODE_IP_ADDR"),
		APIAddr:          os.Getenv("NODE_API_ADDR"),
		GatewayAddr:      os.Getenv("NODE_GATEWAY_ADDR"),
		StorageDirectory: os.Getenv("NODE_STORAGE_DIRECTORY"),
		RuntimeBinary:    os.Getenv("NODE_RUNTIME_BINARY"),
		VolumeGroup:      os.Getenv("NODE_VOLUME_GROUP"),
		ControlPlaneURL:  os.Getenv("NODE_CONTROL_PLANE_URL"),
	}
	if config.APIAddr == "" {
		config.APIAddr = ":3443"
	}
	if config.GatewayAddr == "" {
		config.GatewayAddr = ":8443"
	}
	if config.StorageDirectory == "" {
		config.StorageDirectory = "/var/lib/baepo"
	}
	if config.VolumeGroup == "" {
		config.VolumeGroup = "vg_baepo"
	}
	if config.ControlPlaneURL == "" {
		config.ControlPlaneURL = "https://api.baepo.cloud"
	}
	if config.ClusterID == "" {
		return nil, errors.New("NODE_CLUSTER_ID env variable required")
	}
	if config.BootstrapToken == "" {
		return nil, errors.New("NODE_BOOTSTRAP_TOKEN env variable required")
	}
	if config.RuntimeBinary == "" {
		return nil, errors.New("NODE_RUNTIME_BINARY env variable required")
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

func provideGORM(config *types.Config) (*gorm.DB, error) {
	dbName := "node.db?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL"
	db, err := gorm.Open(sqlite.Open(path.Join(config.StorageDirectory, dbName)), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&types.Volume{},
		&types.Image{},
		&types.NetworkInterface{},
		&types.Machine{},
		&types.MachineEvent{},
		&types.MachineVolume{},
		&types.Container{},
	)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func provideControlPlaneApiClient(config *types.Config) apiv1pbconnect.NodeControllerServiceClient {
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
	return apiv1pbconnect.NewNodeControllerServiceClient(client, config.ControlPlaneURL)
}
