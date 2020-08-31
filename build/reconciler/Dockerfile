# Build the reconciler binary
FROM golang:1.14 as builder

WORKDIR /workspace

COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o reconciler -mod=vendor cmd/reconciler/main.go

FROM k8s.gcr.io/debian-base:v2.0.0
WORKDIR /
COPY --from=builder /workspace/reconciler .
RUN apt-get update && apt-get install -y git

# License file required for on-prem release.
COPY LICENSE LICENSE

# Copy in golang-lru source (MPL licensed dependency)
COPY vendor/github.com/hashicorp/golang-lru/ golang-lru/

# Set up a HOME directory for non-root user
RUN mkdir -p /nomos
RUN chown 1000 /nomos
ENV HOME="/nomos"

# Switch to non-root user
USER 1000

ENTRYPOINT ["/reconciler"]
