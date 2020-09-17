# Build all nomos go binaries
FROM golang:1.14 as bins

WORKDIR /workspace

COPY . .

# Build all our stuff.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on \
  go install -mod=vendor ./cmd/reconciler ./cmd/reconciler-manager
