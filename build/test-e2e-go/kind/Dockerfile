FROM golang:1.14.2-alpine

WORKDIR /repo

# Since go modules isn't enabled by default.
ENV GO111MODULE=on
# Build static binaries; otherwise go test complains.
ENV CGO_ENABLED=0
# Set Kubernetes environment.
ENV KUBERNETES_ENV=KIND

# We're running tests within kind v0.10.0 clusters.
# If you upgrade the version, you also need to update the special kubekins-with-Kind
# images we use in the test-infra repository. See Makefile.prow.
RUN go get sigs.k8s.io/kind@v0.10.0

RUN apk add --no-cache \
  bash curl docker gcc git jq make openssh-client

# Install kubectl and make sure it's available in the PATH.
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl
RUN mv ./kubectl /bin

# Install Helm v3.6.1
RUN wget https://get.helm.sh/helm-v3.6.1-linux-amd64.tar.gz -O /tmp/helm-v3.6.1-linux-amd64.tar.gz && \
  tar -zxvf /tmp/helm-v3.6.1-linux-amd64.tar.gz -C /tmp && \
  mv /tmp/linux-amd64/helm /usr/local/bin/helm && \
  rm -rf /tmp/linux-amd64 /tmp/helm-v3.6.1-linux-amd64.tar.gz

# Install Kustomize v4.1.3
RUN wget https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv4.1.3/kustomize_v4.1.3_linux_amd64.tar.gz -O /tmp/kustomize_v4.1.3_linux_amd64.tar.gz && \
  tar -zxvf /tmp/kustomize_v4.1.3_linux_amd64.tar.gz -C /tmp && \
  mv /tmp/kustomize /usr/local/bin/kustomize && \
  rm /tmp/kustomize_v4.1.3_linux_amd64.tar.gz

# Get go-junit-report.
RUN go get -u github.com/jstemmer/go-junit-report

# Steps after here can't be cached since they touch the local filesystem.

# Just copy everything in the nomos repository so we have whatever we might need.
COPY . .

# Make sure the nomos command is available for tests.
RUN go install github.com/google/nomos/cmd/nomos

# We want bats to be in PATH rather than having to alias it.
COPY third_party/bats-core/bin/bats /bin
