# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.27
FROM golang:${GO_VERSION}-alpine3.24@sha256:738d1cf061836894ff6bb8c33881080ac66de8cf0586615012a0c8f592649cfa AS builder

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
    go build -ldflags="-s -w" -o /out/operator ./cmd/operator

FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6
RUN apk upgrade --no-cache
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 kafscale
USER 10001
WORKDIR /app

COPY --from=builder /out/operator /usr/local/bin/kafscale-operator

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/kafscale-operator"]
