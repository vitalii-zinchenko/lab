# Jsonnet renderer — installs go-jsonnet and jb, then renders dashboards.
# Used only by `make dashboards`; not deployed anywhere.

FROM golang:alpine

RUN apk add --no-cache git ca-certificates

RUN go install github.com/google/go-jsonnet/cmd/jsonnet@latest && \
    go install github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@latest

WORKDIR /src
