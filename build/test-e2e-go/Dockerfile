FROM golang:1.14.2-alpine

WORKDIR /repo

# Since go modules isn't enabled by default.
ENV GO111MODULE=on
# Build static binaries; otherwise go test complains.
ENV CGO_ENABLED=0

# We're running tests within kind v0.8.1 clusters.
RUN go get sigs.k8s.io/kind@v0.8.1

RUN apk add --no-cache \
  bash curl docker gcc git jq make openssh-client

# Install kubectl and make sure it's available in the PATH.
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl
RUN mv ./kubectl /bin

# Steps after here can't be cached since they touch the local filesystem.

# Just copy everyhing in the nomos repository so we have whatever we might need.
COPY . .

# Make sure the nomos command is available for tests.
RUN go install github.com/google/nomos/cmd/nomos

# We want bats to be in PATH rather than having to alias it.
COPY third_party/bats-core/bin/bats /bin
