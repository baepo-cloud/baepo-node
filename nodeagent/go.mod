module github.com/baepo-cloud/baepo-node/nodeagent

go 1.24.2

replace github.com/baepo-cloud/baepo-node/core => ../core

tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

require (
	connectrpc.com/connect v1.18.1
	github.com/baepo-cloud/baepo-node/core v0.0.0-00010101000000-000000000000
	github.com/baepo-cloud/baepo-proto/go v0.0.0-20250801083936-d954a25a2156
	github.com/expected-so/canonicallog v0.0.0-20250112093902-7d3bba85c4b9
	github.com/google/go-containerregistry v0.20.3
	github.com/joho/godotenv v1.5.1
	github.com/nrednav/cuid2 v1.0.1
	github.com/shirou/gopsutil/v3 v3.24.5
	github.com/sourcegraph/conc v0.3.0
	github.com/vishvananda/netlink v1.3.0
	go.uber.org/fx v1.23.0
	golang.org/x/net v0.38.0
	golang.org/x/sys v0.32.0
	google.golang.org/protobuf v1.36.6
	gorm.io/driver/sqlite v1.5.7
	gorm.io/gorm v1.25.12
)

require (
	github.com/alphadose/haxmap v1.4.1 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.16.3 // indirect
	github.com/docker/cli v27.5.0+incompatible // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.2 // indirect
	github.com/dprotaso/go-yit v0.0.0-20240618133044-5a0af90af097 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/getkin/kin-openapi v0.131.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.27 // indirect
	github.com/mdlayher/socket v0.5.1 // indirect
	github.com/mdlayher/vsock v1.2.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/oapi-codegen/oapi-codegen/v2 v2.4.1 // indirect
	github.com/oasdiff/yaml v0.0.0-20250309154309-f31be36b4037 // indirect
	github.com/oasdiff/yaml3 v0.0.0-20250309153720-d2182401db90 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/speakeasy-api/jsonpath v0.6.1 // indirect
	github.com/speakeasy-api/openapi-overlay v0.10.1 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/vbatts/tar-split v0.11.6 // indirect
	github.com/vishvananda/netns v0.0.5 // indirect
	github.com/vmware-labs/yaml-jsonpath v0.3.2 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.uber.org/dig v1.18.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/exp v0.0.0-20221031165847-c99f073a8326 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	golang.org/x/tools v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/grpc v1.72.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
