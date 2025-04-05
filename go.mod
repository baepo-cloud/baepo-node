module github.com/baepo-app/baepo-node

go 1.24.1

replace github.com/baepo-app/baepo-oss => ../baepo-oss

tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

require (
	connectrpc.com/connect v1.18.1
	github.com/baepo-app/baepo-oss v0.0.0-20250403204338-b25abe60ae16
	github.com/joho/godotenv v1.5.1
	github.com/mdlayher/vsock v1.2.1
	github.com/nrednav/cuid2 v1.0.1
	github.com/vishvananda/netlink v1.3.0
	go.uber.org/fx v1.23.0
	golang.org/x/net v0.38.0
	golang.org/x/sys v0.31.0
	google.golang.org/protobuf v1.36.6
)

require (
	github.com/dprotaso/go-yit v0.0.0-20240618133044-5a0af90af097 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/getkin/kin-openapi v0.131.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mdlayher/socket v0.4.1 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/oapi-codegen/oapi-codegen/v2 v2.4.1 // indirect
	github.com/oasdiff/yaml v0.0.0-20250309154309-f31be36b4037 // indirect
	github.com/oasdiff/yaml3 v0.0.0-20250309153720-d2182401db90 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/speakeasy-api/jsonpath v0.6.1 // indirect
	github.com/speakeasy-api/openapi-overlay v0.10.1 // indirect
	github.com/vishvananda/netns v0.0.5 // indirect
	github.com/vmware-labs/yaml-jsonpath v0.3.2 // indirect
	go.uber.org/dig v1.18.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	golang.org/x/tools v0.31.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
