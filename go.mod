module fuse_video_steamer

go 1.23.2

require (
	github.com/anacrolix/fuse v0.3.1
	github.com/sushydev/ring_buffer_go v0.1.3
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.67.1
	google.golang.org/protobuf v1.35.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/exp v0.0.0-20241204233417-43b7b7cde48d // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241015192408-796eee8c2d53 // indirect
)

replace github.com/sushydev/ring_buffer_go => ../ring_buffer_go
