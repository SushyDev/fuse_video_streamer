module fuse_video_steamer

go 1.24.0

replace github.com/sushydev/stream_mount_api => ../stream_mount_api

require (
	github.com/anacrolix/fuse v0.3.1
	github.com/sushydev/ring_buffer_go v0.1.8
	github.com/sushydev/stream_mount_api v0.0.0-20250314214840-50d899a6e4fd
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.71.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/exp v0.0.0-20241204233417-43b7b7cde48d // indirect
	golang.org/x/net v0.37.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250313205543-e70fdf4c4cb4 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)
