# Development Guide - Fuse Video Streamer

## Quick Start

### Prerequisites
- Go 1.24.0 or later
- Make (for build automation)
- Docker (optional, for containerized development)
- Git

### Setup Development Environment

1. **Clone and setup:**
   ```bash
   git clone https://github.com/SushyDev/fuse_video_streamer.git
   cd fuse_video_streamer
   make dev-setup  # Install linting and formatting tools
   ```

2. **Install dependencies:**
   ```bash
   make deps
   ```

3. **Build the project:**
   ```bash
   make build
   ```

4. **Run tests:**
   ```bash
   make test
   ```

### Development Workflow

#### Building
```bash
make build          # Development build
make build-release  # Optimized release build
```

#### Testing
```bash
make test           # Run all tests
make test-verbose   # Run tests with verbose output
make test-coverage  # Generate coverage report
```

#### Code Quality
```bash
make fmt           # Format code
make lint          # Run linter
make vet           # Run go vet
```

#### Running the Application
```bash
# Create a config.yml file first (see example below)
make run           # Build and run
make debug         # Run with race detection
```

### Project Structure

```
fuse_video_streamer/
├── app/                          # Main application code
│   ├── main.go                   # Application entry point
│   ├── config/                   # Configuration management
│   ├── filesystem/               # FUSE filesystem implementation
│   │   ├── client/               # gRPC client for file servers
│   │   ├── driver/               # FUSE driver implementation
│   │   └── interfaces/           # Filesystem interfaces
│   ├── stream/                   # HTTP streaming engine
│   │   ├── drivers/              # Stream driver implementations
│   │   └── interfaces/           # Stream interfaces
│   ├── logger/                   # Logging infrastructure
│   ├── metrics/                  # Metrics collection (WIP)
│   └── testutil/                 # Testing utilities
├── build/                        # Build scripts and configs
├── docs/                         # Documentation
├── Dockerfile                    # Container definition
├── Makefile                      # Build automation
└── .golangci.yml                 # Linter configuration
```

### Configuration

Create a `config.yml` file in the app directory:

```yaml
mount_point: "/mnt/fvs"
volume_name: "fvs"
file_servers:
  - name: "primary_server"
    target: "your-server.com:9090"
  - name: "backup_server"
    target: "backup-server.com:9090"
```

### Development Container (Optional)

For consistent development environment:

```bash
# Using VS Code Dev Containers
code .  # Open in VS Code and reopen in container when prompted

# Or manually with Docker
docker build -t fvs-dev -f Dockerfile.dev .
docker run -it --rm -v $(pwd):/workspace fvs-dev
```

### Testing Guidelines

#### Unit Tests
- Test files should be named `*_test.go`
- Use table-driven tests where applicable
- Mock external dependencies using `testutil` package
- Aim for >80% code coverage

Example test structure:
```go
func TestConfigValidation(t *testing.T) {
    tests := []struct {
        name        string
        config      string
        expectError bool
        errorMsg    string
    }{
        {
            name: "valid config",
            config: `mount_point: "/tmp"...`,
            expectError: false,
        },
        // more test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

#### Integration Tests
- Test FUSE mount/unmount operations
- Test gRPC client connections
- Test end-to-end streaming workflows

#### Test Utilities
Use the `testutil` package for common testing needs:
```go
// Create temporary config
configDir := testutil.CreateTempConfig(t, testutil.TestConfig{
    MountPoint: "/tmp/test",
    VolumeName: "test",
    FileServers: []testutil.TestFileServer{
        {Name: "test", Target: "localhost:9090"},
    },
})

// Create mock gRPC server
mockServer := testutil.NewMockGRPCServer()
defer mockServer.Stop()
```

### Debugging

#### Enable Debug Mode
```bash
# Uncomment debug line in main.go
go run main.go  # Will start pprof server on :6060
```

#### Profiling
```bash
# CPU profiling
go tool pprof http://localhost:6060/debug/pprof/profile

# Memory profiling
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profiling
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

#### Logging
The application uses structured logging with Zap. Log levels can be controlled through the logger configuration.

### Common Development Tasks

#### Adding New Tests
1. Create `*_test.go` file in the same package
2. Use `testutil` helpers for common setup
3. Run tests: `make test`
4. Check coverage: `make test-coverage`

#### Adding New Features
1. Define interfaces in appropriate `interfaces` package
2. Implement feature with proper error handling
3. Add comprehensive tests
4. Update documentation

#### Performance Optimization
1. Use benchmarks: `make benchmark`
2. Profile with pprof
3. Monitor metrics (when implemented)
4. Load test with realistic scenarios

### Contributing

1. **Code Style**: Follow Go conventions, use `make fmt` and `make lint`
2. **Testing**: Add tests for new features, maintain >80% coverage
3. **Documentation**: Update docs for public APIs and significant changes
4. **Commits**: Use conventional commit messages

### Troubleshooting

#### Build Issues
- Ensure Go 1.24.0+ is installed
- Run `make deps` to install dependencies
- Check for compilation errors in `make build`

#### Test Failures
- Run `make test-verbose` for detailed output
- Check for race conditions with `make debug`
- Verify test isolation (no shared state)

#### Runtime Issues
- Check FUSE permissions (`CAP_SYS_ADMIN`, `/dev/fuse` access)
- Verify gRPC server connectivity
- Check configuration file format and paths
- Review application logs for errors

#### Performance Issues
- Enable profiling and analyze bottlenecks
- Check network latency to file servers
- Monitor memory usage and buffer sizes
- Review concurrent access patterns