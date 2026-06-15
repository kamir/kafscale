# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26
FROM golang:${GO_VERSION}-alpine@sha256:7a3e50096189ad57c9f9f865e7e4aa8585ed1585248513dc5cda498e2f41812c AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /out/proxy ./cmd/proxy

FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 kafscale
USER 10001
WORKDIR /app

COPY --from=builder /out/proxy /usr/local/bin/kafscale-proxy

EXPOSE 9092
ENTRYPOINT ["/usr/local/bin/kafscale-proxy"]
