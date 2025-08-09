FROM nixos/nix:latest AS app

RUN nix-env -iA nixpkgs.go nixpkgs.git

ENV GO111MODULE=on \
    GOPROXY=direct \
    GOFLAGS=-mod=readonly \
    GOTOOLCHAIN=go1.24.0+auto

WORKDIR /src/app

COPY app/go.mod app/go.sum ./
RUN go mod download

COPY app .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -extldflags '-static'" -o /out/main main.go

FROM nixos/nix:latest AS dependencies

RUN mkdir -p /root/.config/nix && \
    echo "experimental-features = nix-command flakes" > /root/.config/nix/nix.conf

WORKDIR /src

COPY . .

# Build the flake.
# 'nix build' will find the flake in ./build and execute it.
# The output is a symlink named 'result' pointing to a directory in the Nix store
# that contains the 'bin', 'lib', and 'etc' folders we created in the flake.
RUN nix build ./build --out-link /out

# --- Final minimal image ---
FROM scratch

ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

ENV PATH=/bin

WORKDIR /app

COPY --from=app /out/main /app/main
COPY --from=dependencies /out/etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=dependencies /out/bin/fusermount /bin/fusermount
COPY --from=dependencies /out/lib /lib

# Run the application directly (no shell available in scratch)
ENTRYPOINT ["/app/main"]
