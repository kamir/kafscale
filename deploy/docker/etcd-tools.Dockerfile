# syntax=docker/dockerfile:1.7
# etcd maintenance tools (etcdctl, etcdutl) for the kafSCALE operator snapshot
# and defrag jobs. Built FROM SOURCE at a pinned etcd tag with bumped
# x/crypto / x/net / grpc: the prebuilt gcr.io/etcd-development/etcd image
# vendors CVE-affected versions of those into its binaries (R2.3). Bumping the
# prebuilt-image tag alone (v3.6.8 -> v3.6.11) left 9 Criticals; building the
# two binaries from source with the deps bumped clears them (Critical=0).
ARG GO_VERSION=1.27
ARG ETCD_VERSION=v3.6.11
FROM golang:${GO_VERSION}-alpine3.24@sha256:738d1cf061836894ff6bb8c33881080ac66de8cf0586615012a0c8f592649cfa AS build
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

FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6
RUN apk upgrade --no-cache
RUN apk add --no-cache ca-certificates
COPY --from=build /out/etcdctl /usr/local/bin/etcdctl
COPY --from=build /out/etcdutl /usr/local/bin/etcdutl
