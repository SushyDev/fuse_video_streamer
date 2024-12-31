<div align="center">
  <p>Fuse file server tailor made for video streaming with gRPC api</p>
</div>

---

## Table of Contents

- [What is Fuse Video Streamer](#what-is-fuse-video-streamer)
- [Getting Started](#getting-started)
  - [Docker Setup](#docker-setup)
  - [Native Setup](#native-setup)
- [Development](#development)
  - [Prerequisites](#prerequisites)
  - [Setup Instructions](#setup-instructions)
  - [Running the Project](#running-the-project)
- [Contributing](#contributing)
- [License](#license)

---

## What is Fuse Video Streamer

Fuse Video Streamer creates a virtual disk sort of like Rclone that contains files and folders. In this case it supports streaming HTTP partial content. Fuse Video Streamer is a completely stateless application, it does not contain any files or directories by itself, instead it will connect to a list of fileservers (provided over the gPRCprotocol) to list out directories and files. When you open a file it will make a final request to get the streamable media url and it will continuesly make HTTP partial content requests to it and stream the file.

## Getting Started

### Docker Setup

The image for this project is available on Docker at `ghcr.io/sushydev/fuse_video_streamer:latest`. Below is an example of a `docker-compose.yml` file to set up the project:

```yaml
fuse_video_streamer:
  container_name: fuse_video_streamer
  image: ghcr.io/sushydev/fuse_video_streamer:latest
  restart: unless-stopped
  network_mode: host  # Preferable if using a specific network
  volumes:
    - ./fuse_video_streamer.yml:/app/config.yml  # Bind configuration
    - ./fvs:/mnt/fvs:rshared # Bind the mount
    - ./logs/fuse_video_streamer:/app/logs  # Store logs
  cap_add:
    - SYS_ADMIN
  security_opt:
    - apparmor:unconfined
  devices:
    - /dev/fuse:/dev/fuse:rwm
  pid: host # Very important, each stream is opened per PID!
```

### Native Setup

To build the project manually, you can use the following Go commands:

1. **Download Dependencies:**
    ```sh
    go mod download
    ```

2. **Build the Project:**
    ```sh
    CGO_ENABLED=0 GOOS=linux go build -o fuse_video_streamer main.go
    ```

### Configuration

Fuse Video Streamer uses a `config.yml` file (Very important its `yml` and not `yaml`) with the following properties

Example `config.yml`.
```yaml
mount_points: "/mnt/fvs"
volume_name: "fvs"
file_servers:
  - "localhost:xxxx"
```

#### Done
Now you're ready to use it
    
---

## Development

### Prerequisites

Ensure you have the following installed on your system:

- **Go** (version 1.23.2 or later)

### Setup Instructions

1. **Install Dependencies:**
    ```sh
    go mod download
    ```

### Running the Project

- **Start:**
    ```sh
    go run main.go
    ```

---

## Contributing

Contributions are welcome! Please follow the guidelines in the [CONTRIBUTING.md](CONTRIBUTING.md) for submitting changes.

---

## License

This project is licensed under the GNU GPLv3 License. See the [LICENSE](LICENSE) file for details.
