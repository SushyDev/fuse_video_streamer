# Dockerfile
FROM golang:1.21-alpine

# Install necessary packages
RUN apk update && apk add --no-cache \
    build-base \
    fuse \
#    libfuse-dev \
    bash \
    git

# Set environment variables
ENV GO111MODULE=on
ENV GOPROXY=direct
ENV GOFLAGS="-mod=readonly"

# Create app directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN go build -o fuse-app main.go

RUN ln -s /bin/fusermount /bin/fusermount3

COPY cert.pem /usr/local/share/ca-certificates/proxyman-ca.crt
RUN update-ca-certificates

# Set entrypoint
ENTRYPOINT ["/app/fuse-app"]
