# syntax=docker/dockerfile:1.7
# etcd maintenance tools (etcdctl, etcdutl) for the kafSCALE operator snapshot
# and defrag jobs. Built FROM SOURCE at a pinned etcd tag with bumped
# x/crypto / x/net / grpc: the prebuilt gcr.io/etcd-development/etcd image
# vendors CVE-affected versions of those into its binaries (R2.3). Bumping the
# prebuilt-image tag alone (v3.6.8 -> v3.6.11) left 9 Criticals; building the
# two binaries from source with the deps bumped clears them (Critical=0).
ARG GO_VERSION=1.26
ARG ETCD_VERSION=v3.6.11
FROM golang:${GO_VERSION}-alpine@sha256:7a3e50096189ad57c9f9f865e7e4aa8585ed1585248513dc5cda498e2f41812c AS build
ARG ETCD_VERSION
ARG TARGETOS
ARG TARGETARCH
RUN apk add --no-cache git
RUN git clone --depth 1 -b ${ETCD_VERSION} https://github.com/etcd-io/etcd /src
ENV GOFLAGS=-mod=mod
WORKDIR /src/etcdctl
RUN go get golang.org/x/crypto@v0.52.0 golang.org/x/net@v0.55.0 google.golang.org/grpc@v1.79.3 && \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o /out/etcdctl .
WORKDIR /src/etcdutl
RUN go get golang.org/x/crypto@v0.52.0 golang.org/x/net@v0.55.0 google.golang.org/grpc@v1.79.3 && \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o /out/etcdutl .

FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d
RUN apk add --no-cache ca-certificates
COPY --from=build /out/etcdctl /usr/local/bin/etcdctl
COPY --from=build /out/etcdutl /usr/local/bin/etcdutl
