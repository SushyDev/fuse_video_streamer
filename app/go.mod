module fuse_video_streamer

go 1.25.4

require (
	github.com/BurntSushi/toml v1.5.0
	github.com/anacrolix/fuse v0.4.0
	github.com/google/uuid v1.6.0
	github.com/sushydev/ring_buffer_go v0.1.9
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.77.0
	sushydev.github.io/stream_mount_api/go v0.0.0-20251207124320-a0ace8ebc642
)

// replace github.com/sushydev/ring_buffer_go => ../../ring_buffer_go

require (
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20241204233417-43b7b7cde48d // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)
