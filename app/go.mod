module fuse_video_streamer

go 1.26.2

require (
	github.com/BurntSushi/toml v1.5.0
	github.com/anacrolix/fuse v0.3.2
	github.com/google/uuid v1.6.0
	github.com/sushydev/ring_buffer_go v0.1.10
	go.uber.org/goleak v1.3.0
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.77.0
	sushydev.github.io/stream_mount_api/go v0.0.0-20251207124320-a0ace8ebc642
)

// replace github.com/sushydev/ring_buffer_go => ../../../ring_buffer_go

require (
	github.com/anacrolix/envpprof v1.4.1-0.20251201125402-e8b52d50f714 // indirect
	github.com/anacrolix/generics v0.1.0 // indirect
	github.com/anacrolix/log v0.14.1 // indirect
	github.com/anacrolix/missinggo v1.2.1 // indirect
	github.com/anacrolix/missinggo/perf v1.0.0 // indirect
	github.com/anacrolix/missinggo/v2 v2.10.0 // indirect
	github.com/anacrolix/sync v0.6.0 // indirect
	github.com/felixge/fgprof v0.9.5 // indirect
	github.com/google/pprof v0.0.0-20240227163752-401108e1b7e7 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20241204233417-43b7b7cde48d // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)
