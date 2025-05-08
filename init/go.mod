module github.com/baepo-cloud/baepo-node/init

go 1.24.2

replace github.com/baepo-cloud/baepo-node/core => ../core

require (
	connectrpc.com/connect v1.18.1
	github.com/baepo-cloud/baepo-node/core v0.0.0-00010101000000-000000000000
	github.com/baepo-cloud/baepo-proto/go v0.0.0-20250508080114-4126d1786353
	github.com/vishvananda/netlink v1.3.0
	golang.org/x/sys v0.32.0
	google.golang.org/protobuf v1.36.6
)

require (
	github.com/mdlayher/socket v0.4.1 // indirect
	github.com/mdlayher/vsock v1.2.1 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
)
