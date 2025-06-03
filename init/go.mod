module github.com/baepo-cloud/baepo-node/init

go 1.24.2

replace github.com/baepo-cloud/baepo-node/core => ../core

require (
	connectrpc.com/connect v1.18.1
	github.com/baepo-cloud/baepo-node/core v0.0.0-00010101000000-000000000000
	github.com/baepo-cloud/baepo-proto/go v0.0.0-20250603063731-92a2a28476d8
	github.com/nrednav/cuid2 v1.0.1
	github.com/vishvananda/netlink v1.3.0
	golang.org/x/sys v0.32.0
	google.golang.org/protobuf v1.36.6
)

require (
	github.com/mdlayher/socket v0.4.1 // indirect
	github.com/mdlayher/vsock v1.2.1 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/grpc v1.72.1 // indirect
)
