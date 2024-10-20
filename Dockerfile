FROM golang:1.23.2-alpine AS builder

RUN apk update && apk add --no-cache \
    build-base \
    fuse \
    bash \
    git \
 && apk add --no-cache --virtual .build-deps \
    gcc \
    musl-dev \
 && apk add --no-cache fuse-dev \
 && rm -rf /var/cache/apk/*

ENV GO111MODULE=on
ENV GOPROXY=direct
ENV GOFLAGS="-mod=readonly"

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o fuse-app main.go

FROM alpine:latest

RUN apk add --no-cache fuse

COPY --from=builder /app/fuse-app /app/fuse-app

RUN chmod +x /app/fuse-app
RUN chmod -R 755 /app

ENTRYPOINT ["/app/fuse-app"]
