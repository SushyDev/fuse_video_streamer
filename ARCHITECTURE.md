# Architecture Documentation - Fuse Video Streamer

## System Overview

Fuse Video Streamer is a sophisticated virtual filesystem that enables seamless streaming of video content from remote servers through a local FUSE mount point. The system is designed with a modular, layered architecture that separates concerns and enables extensibility.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                 User Applications                        │
│            (VLC, mpv, Plex, etc.)                      │
└─────────────────┬───────────────────────────────────────┘
                  │ File System Calls
                  │ (open, read, seek, etc.)
┌─────────────────▼───────────────────────────────────────┐
│                FUSE Layer                                │
│         (Virtual File System Interface)                 │
└─────────────────┬───────────────────────────────────────┘
                  │ FUSE Operations
                  │
┌─────────────────▼───────────────────────────────────────┐
│            Fuse Video Streamer                          │
│                                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │   Config    │  │ Filesystem  │  │  Streaming  │     │
│  │ Management  │  │   Layer     │  │   Engine    │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
│                           │                            │
│  ┌─────────────┐  ┌───────▼──────┐  ┌─────────────┐     │
│  │  Logging &  │  │  gRPC Client │  │   Metrics   │     │
│  │  Metrics    │  │    Layer     │  │ Collection  │     │
│  └─────────────┘  └──────────────┘  └─────────────┘     │
└─────────────────┬───────────────────────────────────────┘
                  │ gRPC Calls
                  │
┌─────────────────▼───────────────────────────────────────┐
│               Remote File Servers                       │
│          (stream_mount_api compatible)                  │
└─────────────────┬───────────────────────────────────────┘
                  │ HTTP Requests
                  │
┌─────────────────▼───────────────────────────────────────┐
│              Content Sources                            │
│        (Cloud Storage, CDNs, HTTP Servers)             │
└─────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Configuration Management (`app/config/`)

**Responsibility**: Centralized configuration loading and validation

**Key Features**:
- YAML-based configuration
- Runtime validation
- Environment-specific settings
- Multi-server support

**Interfaces**:
```go
type Config struct {
    MountPoint  string               `yaml:"mount_point"`
    VolumeName  string               `yaml:"volume_name"`
    FileServers []FileSystemProvider `yaml:"file_servers"`
}
```

### 2. FUSE Filesystem Layer (`app/filesystem/`)

**Responsibility**: Virtual filesystem implementation using FUSE

**Architecture**:
```
filesystem/
├── driver/
│   ├── provider/fuse/        # FUSE-specific implementation
│   │   ├── internal/
│   │   │   ├── filesystem/   # Node types (file, directory, symlink)
│   │   │   ├── api/          # FUSE API integration
│   │   │   ├── pool/         # Resource pooling
│   │   │   ├── registry/     # Node registry
│   │   │   └── server/       # FUSE server
│   │   └── metrics/          # FUSE-specific metrics
│   └── service/              # Service layer abstraction
├── client/                   # gRPC client for remote servers
└── interfaces/               # Common interfaces
```

**Key Components**:

#### Node Types
- **Root Node**: Filesystem root, lists file servers as subdirectories
- **Directory Node**: Represents directories, handles listing operations
- **File Node**: Regular files, delegates to streamable handle for large files
- **Streamable Node**: Optimized for video files with streaming support
- **Symlink Node**: Symbolic link support

#### Handle Types
- **Directory Handle**: Manages directory reading and metadata
- **File Handle**: Standard file operations
- **Streamable Handle**: Optimized streaming operations with HTTP range support

### 3. gRPC Client Layer (`app/filesystem/client/`)

**Responsibility**: Communication with remote file servers

**Features**:
- Connection pooling and management
- Automatic failover between servers
- Metadata caching (planned)
- Error handling and retry logic

**Protocol**: Uses `stream_mount_api` for:
- Directory listing (`ListDirectory`)
- File metadata (`GetFileInfo`)
- Stream URL resolution (`GetStreamUrl`)

### 4. HTTP Streaming Engine (`app/stream/`)

**Responsibility**: Efficient video streaming with HTTP range requests

**Architecture**:
```
stream/
├── drivers/
│   └── http_ring_buffer/     # Ring buffer-based HTTP streaming
│       ├── internal/
│       │   ├── connection/   # HTTP connection management
│       │   ├── transfer/     # Data transfer coordination
│       │   └── pool.go       # Stream pooling
│       └── factory/          # Stream factory
└── interfaces/               # Stream interfaces
```

**Key Features**:
- **Ring Buffer**: Circular buffer for efficient memory usage
- **Adaptive Buffering**: Dynamic buffer sizing based on file size
- **Preloading**: Intelligent read-ahead for smooth playback
- **HTTP Range Support**: Partial content requests for seeking
- **Concurrent Streams**: Multiple simultaneous video streams

**Buffer Sizing Strategy**:
```
File Size    | Buffer Size | Preload Size
< 1GB        | 64MB        | 32MB
1-10GB       | 256MB       | 128MB
10-50GB      | 512MB       | 256MB
> 50GB       | 1GB         | 16MB (max)
```

### 5. Logging Infrastructure (`app/logger/`)

**Responsibility**: Structured logging throughout the application

**Features**:
- Zap-based structured logging
- Configurable log levels
- Component-specific loggers
- Performance optimized

### 6. Metrics Collection (`app/metrics/`)

**Responsibility**: Performance monitoring and observability

**Metrics Categories**:
- Filesystem operations (latency, throughput)
- Stream performance (buffer efficiency, cache hits)
- gRPC call statistics
- Error rates and types

## Data Flow

### File Access Flow

1. **User Application** makes filesystem call (e.g., `open("/mnt/fvs/server1/video.mp4")`)
2. **FUSE Layer** intercepts the call and routes to appropriate node
3. **Filesystem Layer** determines if file exists via gRPC call to server
4. **gRPC Client** queries remote server for file metadata
5. **Streamable Node** creates streamable handle for video files
6. **Streaming Engine** initializes HTTP connection and ring buffer
7. **Data Returns** through layers back to user application

### Streaming Read Flow

1. **User Application** requests data at specific offset (`read()` at position X)
2. **Streamable Handle** checks if data available in ring buffer
3. **If Not Available**:
   - Calculate optimal request range (with preloading)
   - Issue HTTP range request to content server
   - Stream data into ring buffer
4. **Return Data** from buffer to application
5. **Background Prefetching** continues to fill buffer

### Directory Listing Flow

1. **User Application** lists directory (`ls /mnt/fvs/server1/`)
2. **Directory Node** checks local cache (if implemented)
3. **gRPC Client** calls `ListDirectory` on remote server
4. **Results Processed** and returned as filesystem entries
5. **Cache Updated** for subsequent requests

## Scalability Considerations

### Horizontal Scaling
- **Multiple File Servers**: Configure multiple backend servers for load distribution
- **Server Failover**: Automatic failover when servers become unavailable
- **Regional Distribution**: Place servers close to content sources

### Vertical Scaling
- **Memory**: Larger ring buffers for better caching
- **CPU**: Parallel stream processing
- **Network**: Multiple concurrent connections per stream

### Performance Optimizations

#### Current Optimizations
- Ring buffer architecture for memory efficiency
- HTTP range requests for bandwidth optimization
- Concurrent stream support
- Adaptive buffer sizing

#### Planned Optimizations
- Metadata caching layer
- Connection pooling
- Read-ahead prediction algorithms
- Compression support

## Security Model

### Current Security
- Filesystem permissions through FUSE
- Basic input validation
- Container isolation

### Planned Security Enhancements
- TLS/mTLS for gRPC connections
- Authentication tokens
- Rate limiting
- Input sanitization
- Audit logging

## Error Handling Strategy

### Error Categories
1. **Configuration Errors**: Invalid config, missing files
2. **Network Errors**: Connection failures, timeouts
3. **Protocol Errors**: gRPC failures, invalid responses
4. **Filesystem Errors**: FUSE operation failures
5. **Resource Errors**: Memory allocation, buffer overflow

### Error Recovery
- **Retry Logic**: Exponential backoff for transient failures
- **Graceful Degradation**: Continue operation with reduced functionality
- **Circuit Breaker**: Prevent cascade failures
- **Failover**: Switch to backup servers when primary fails

## Monitoring and Observability

### Current Monitoring
- Structured logging with Zap
- Basic performance metrics
- Debug profiling support

### Planned Monitoring
- Prometheus metrics export
- Distributed tracing
- Health check endpoints
- Performance dashboards

## Extension Points

### Adding New Filesystem Providers
1. Implement `FileSystemServerService` interface
2. Add provider-specific configuration
3. Register in service factory

### Adding New Stream Drivers
1. Implement `Stream` interface
2. Add driver-specific optimizations
3. Register in stream factory

### Adding New Protocols
1. Define new client interface
2. Implement protocol-specific client
3. Update filesystem layer to use new client

## Dependencies

### Core Dependencies
- **FUSE**: `github.com/anacrolix/fuse` - Filesystem interface
- **gRPC**: `google.golang.org/grpc` - Remote communication
- **Ring Buffer**: `github.com/sushydev/ring_buffer_go` - Memory management
- **Logging**: `go.uber.org/zap` - Structured logging
- **Config**: `gopkg.in/yaml.v3` - Configuration parsing

### Protocol Dependencies
- **Stream Mount API**: `github.com/sushydev/stream_mount_api` - gRPC protocol definitions

## Future Architecture Considerations

### Microservices Decomposition
- Separate metadata service
- Dedicated streaming service
- Configuration service
- Metrics aggregation service

### Event-Driven Architecture
- Async event processing
- Stream lifecycle events
- Performance metric events
- Error notification events

### Caching Strategy
- Multi-level caching (memory, disk, distributed)
- Cache invalidation policies
- Cache warming strategies
- Cache consistency guarantees