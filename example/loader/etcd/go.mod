module trpc.group/trpc-go/trpc-gateway/example

go 1.18

require (
	trpc.group/trpc-go/trpc-gateway v0.0.5
	trpc.group/trpc-go/trpc-gateway/core/loader/etcd v0.0.1
	trpc.group/trpc-go/trpc-go v0.0.0-20231008070952-27a655b3e79c
)

require (
	code.cloudfoundry.org/bytefmt v0.0.0-20211005130812-5bb3c17173e5 // indirect
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-playground/form/v4 v4.2.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v2.0.0+incompatible // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.16.3 // indirect
	github.com/lestrrat-go/strftime v1.0.6 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/panjf2000/ants/v2 v2.4.7 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.45.0 // indirect
	go.etcd.io/etcd/api/v3 v3.5.9 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.9 // indirect
	go.etcd.io/etcd/client/v3 v3.5.9 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/automaxprocs v1.3.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/grpc v1.51.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	trpc.group/trpc-go/tnet v0.0.0-20230810071536-9d05338021cf // indirect
	trpc.group/trpc-go/trpc-config-etcd v0.0.0-20230829073930-07f202f52c32 // indirect
	trpc.group/trpc/trpc-protocol/pb/go/trpc v0.0.0-20230803031059-de4168eb5952 // indirect
)

replace (
	trpc.group/trpc-go/trpc-gateway v0.0.5 => ../../../
	trpc.group/trpc-go/trpc-gateway/core/loader/etcd v0.0.1 => ./../../../core/loader/etcd
)
