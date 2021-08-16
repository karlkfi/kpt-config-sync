# Build intermediate image for gcloud / kubectl
FROM gcr.io/google.com/cloudsdktool/cloud-sdk:slim as gcloud-install
RUN apt-get install -y kubectl

# Build e2e image
FROM golang:1.14.2-alpine

WORKDIR /repo

# Since go modules isn't enabled by default.
ENV GO111MODULE=on
# Build static binaries; otherwise go test complains.
ENV CGO_ENABLED=0
# Set Kubernetes environment.
ENV KUBERNETES_ENV=GKE

RUN apk add bash curl docker gcc git jq make openssh-client python

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

# Copy installed gcloud and kubectl into image
COPY --from=gcloud-install /usr/lib/google-cloud-sdk /opt/gcloud/google-cloud-sdk
COPY --from=gcloud-install /usr/bin/kubectl /opt/gcloud/google-cloud-sdk/bin/kubectl
ENV PATH /opt/gcloud/google-cloud-sdk/bin:$PATH

# Get go-junit-report
RUN go get -u github.com/jstemmer/go-junit-report

# Steps after here can't be cached since they touch the local filesystem.

# Just copy everything in the nomos repository so we have whatever we might need.
COPY . .

# Make sure the nomos command is available for tests.
RUN go install github.com/google/nomos/cmd/nomos

# We want bats to be in PATH rather than having to alias it.
COPY third_party/bats-core/bin/bats /bin
