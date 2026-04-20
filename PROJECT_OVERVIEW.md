# Fuse Video Streamer - Project Overview

## What is Fuse Video Streamer?

Fuse Video Streamer is a sophisticated Go-based application that creates a virtual file system using FUSE (Filesystem in Userspace) specifically optimized for video streaming. It acts as a bridge between remote file servers and local applications, providing seamless access to remote media files through a native filesystem interface.

## Core Architecture

### High-Level Design
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Local Apps    │◄──►│  FUSE Virtual    │◄──►│   Remote File   │
│  (Video Players)│    │   File System    │    │    Servers      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │                         │
                              ▼                         ▼
                       ┌─────────────┐          ┌─────────────┐
                       │   HTTP      │          │    gRPC     │
                       │  Streaming  │          │  Protocol   │
                       └─────────────┘          └─────────────┘
```

### Key Components

1. **FUSE Driver** (`app/filesystem/driver/provider/fuse/`)
   - Implements FUSE filesystem operations
   - Handles file/directory listing, reading, and metadata
   - Manages virtual filesystem state

2. **gRPC Client** (`app/filesystem/client/`)
   - Communicates with remote file servers
   - Uses `stream_mount_api` protocol for file operations
   - Handles server discovery and connection management

3. **HTTP Streaming Engine** (`app/stream/`)
   - Implements HTTP partial content requests (Range headers)
   - Optimizes video streaming with ring buffer architecture
   - Handles concurrent read operations

4. **Configuration Management** (`app/config/`)
   - YAML-based configuration system
   - Validates mount points and server connections
   - Supports multiple file server backends

5. **Logging & Metrics** (`app/logger/`, `app/metrics/`)
   - Structured logging with Zap
   - Performance metrics collection
   - Debug and profiling capabilities

## Technical Specifications

- **Language**: Go 1.24.0
- **Total Code**: ~4,662 lines across 57 files
- **Key Dependencies**:
  - `github.com/anacrolix/fuse` - FUSE filesystem implementation
  - `github.com/sushydev/stream_mount_api` - gRPC API for file servers
  - `github.com/sushydev/ring_buffer_go` - Optimized streaming buffer
  - `go.uber.org/zap` - Structured logging
  - `google.golang.org/grpc` - gRPC communication

## Current Features

### ✅ Working Features
- Virtual filesystem mounting via FUSE
- gRPC communication with remote file servers
- HTTP partial content streaming for video files
- Docker containerization with multi-architecture support
- Graceful shutdown and signal handling
- Structured logging and basic metrics
- CI/CD pipeline with GitHub Actions

### 🚧 Limitations
- No test coverage (0% currently)
- Limited error handling and recovery
- Basic configuration validation
- No caching layer implementation
- Limited monitoring and observability
- No security features (TLS, authentication)

## Performance Characteristics

- **Stateless Design**: No local file storage, pure streaming
- **Ring Buffer Architecture**: Optimized for video streaming workloads
- **Concurrent Operations**: Supports multiple simultaneous streams
- **Memory Efficient**: Minimal memory footprint per stream

## Deployment Options

### Docker (Recommended)
```yaml
fuse_video_streamer:
  image: ghcr.io/sushydev/fuse_video_streamer:latest
  volumes:
    - ./config.yml:/app/config.yml
    - ./fvs:/mnt/fvs:rshared
  cap_add:
    - SYS_ADMIN
  devices:
    - /dev/fuse:/dev/fuse:rwm
  pid: host
```

### Native Build
```bash
go mod download
CGO_ENABLED=0 GOOS=linux go build -o fuse_video_streamer main.go
```

## Configuration Example

```yaml
mount_point: "/mnt/fvs"
volume_name: "fvs"
file_servers:
  - name: "primary_server"
    target: "fileserver.example.com:9090"
  - name: "backup_server"
    target: "backup.example.com:9090"
```

## Use Cases

1. **Media Streaming**: Stream large video files without local storage
2. **Cloud Integration**: Access cloud-stored media through filesystem
3. **Distributed Storage**: Aggregate multiple remote storage backends
4. **Bandwidth Optimization**: Stream only requested file portions

## Future Roadmap

The project is well-architected but needs significant improvements in testing, documentation, and operational capabilities. See the detailed improvement plan below.