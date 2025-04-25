# lazy-proxy
`lazy-proxy` is a lightweight TCP proxy service which starts another service process on demand and stops it after a configurable timeout. It handles WebSockets making it suitable for most HTTP workloads.

This project was inspired by the need to emulate IIS's 20-minute idle timeout for App Pools, helping conserve memory by stopping unused services and waking them up on demand.

## Building using docker
`docker build -t lazy-proxy-build .` or use `build.sh` to build and copy executable to the current directory

## Building using go
`go build -ldflags="-s -w" -o lazy-proxy .`

## Usage (for /app/my-service)

`./lazy-proxy -idle 20m -listen :8050 -forward 127.0.0.1:8080 /app/my-service`

## Example Dockerfile for a dotnet application (for /app/dotnet/My-App.dll)

### Generic Dockerfile based on a dotnet 8 image with lazy-proxt prebuilt:

```
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY lazy-proxy/* .
RUN go build -ldflags="-s -w" -o lazy-proxy .

FROM mcr.microsoft.com/dotnet/aspnet:8.0-noble

# Set working directory
WORKDIR /app

COPY --from=builder /app/lazy-proxy /usr/local/bin/lazy-proxy
```

Build above using: `docker build -t dotnet8:lazy .`

### Compose file using above Docker image with an example configured entrypoint

```
services:
    myapp:
        image: dotnet8:lazy
        working_dir: /app/dotnet
        entrypoint: /usr/local/bin/lazy-proxy -idle 1m -listen :8050 -forward 127.0.0.1:8080 dotnet /app/dotnet/My-App.dll
        volumes:
            - ./my-app:/app/dotnet/
            - /var/lib/dbus/machine-id:/var/lib/dbus/machine-id:ro
```