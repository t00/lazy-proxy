FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY lazy-proxy.go .
COPY go.mod .

RUN go build -ldflags="-s -w" -o lazy-proxy .
