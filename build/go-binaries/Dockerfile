# Build all nomos go binaries
FROM golang:1.14 as bins

WORKDIR /workspace

COPY . .

# Version string to embed in built binary.
ARG VERSION

# Build all our stuff.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on \
  go install \
    -mod=vendor \
    -ldflags "-X github.com/google/nomos/pkg/version.VERSION=${VERSION}" \
    ./cmd/reconciler \
    ./cmd/reconciler-manager
