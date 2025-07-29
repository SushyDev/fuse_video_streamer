FROM golang:1.24.0-alpine AS builder

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

COPY app/go.mod app/go.sum ./

RUN go mod download

COPY app .

RUN CGO_ENABLED=0 GOOS=linux go build -o main main.go

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache fuse su-exec

COPY --from=builder /app/main /app/main

COPY build/entrypoint.sh /app/entrypoint.sh

RUN chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
