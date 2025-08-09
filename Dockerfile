# Builder stage using NixOS to provision toolchains
FROM nixos/nix:latest AS builder

# Install Go toolchain, git, and CA certificates
# We rely on Go's toolchain auto-download to obtain the exact version (1.24.0)
RUN nix-env -iA nixpkgs.go nixpkgs.git nixpkgs.cacert

ENV GO111MODULE=on \
    GOPROXY=direct \
    GOFLAGS=-mod=readonly \
    GOTOOLCHAIN=go1.24.0+auto

WORKDIR /src/app

# Cache module download layer
COPY app/go.mod app/go.sum ./
RUN go mod download

# Build application
COPY app .
# Produce a static Linux binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w -extldflags '-static'" -o /out/main main.go

# Prepare CA bundle for the scratch image
RUN mkdir -p /out/etc/ssl/certs \
 && cp -L $(nix-env -q --out-path cacert | awk '{print $2}')/etc/ssl/certs/ca-bundle.crt /out/etc/ssl/certs/ca-certificates.crt

# Final minimal image
FROM scratch

# Ensure Go's crypto uses the CA bundle
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

WORKDIR /app

# Copy static binary and CA certificates
COPY --from=builder /out/main /app/main
COPY --from=builder /out/etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Run the application directly (no shell available in scratch)
ENTRYPOINT ["/app/main"]
