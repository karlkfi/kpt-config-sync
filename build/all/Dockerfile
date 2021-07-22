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
    ./cmd/reconciler-manager \
    ./cmd/monitor \
    ./cmd/git-importer \
    ./cmd/hydration-controller \
    ./cmd/nomos \
    ./cmd/admission-webhook

# Hydration controller image
FROM k8s.gcr.io/build-image/debian-base-amd64:buster-v1.8.0 as hydration-controller
WORKDIR /
COPY --from=bins /go/bin/hydration-controller .
RUN apt-get update && apt-get install -y git wget

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

# License file required for on-prem release.
COPY LICENSE LICENSE

# Copy in golang-lru source (MPL licensed dependency)
COPY vendor/github.com/hashicorp/golang-lru/ golang-lru/

# Switch to non-root user
USER 1000

ENTRYPOINT ["/hydration-controller"]

# Reconciler image
FROM k8s.gcr.io/build-image/debian-base-amd64:buster-v1.8.0 as reconciler
WORKDIR /
COPY --from=bins /go/bin/reconciler .
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

# Reconciler Manager image
FROM gcr.io/distroless/static:nonroot as reconciler-manager
WORKDIR /
COPY --from=bins /go/bin/reconciler-manager reconciler-manager
USER nonroot:nonroot

# License file required for on-prem release.
COPY LICENSE LICENSE

# Copy in golang-lru source (MPL licensed dependency)
COPY vendor/github.com/hashicorp/golang-lru/ golang-lru/

ENTRYPOINT ["/reconciler-manager"]

# Admission Webhook image
FROM gcr.io/distroless/static:nonroot as admission-webhook
WORKDIR /
COPY --from=bins /go/bin/admission-webhook admission-webhook
USER nonroot:nonroot

# License file required for on-prem release.
COPY LICENSE LICENSE

# Copy in golang-lru source (MPL licensed dependency)
COPY vendor/github.com/hashicorp/golang-lru/ golang-lru/

ENTRYPOINT ["/admission-webhook"]

# Nomos image
FROM k8s.gcr.io/build-image/debian-base-amd64:buster-v1.8.0 as nomos

# https://github.com/GoogleCloudPlatform/google-cloud-go/issues/791#issuecomment-353689746
RUN apt-get update && apt-get install -y \
  libcap2-bin bash git

RUN mkdir -p /opt/nomos/bin
ENV PATH="/opt/nomos/bin:${PATH}"
WORKDIR /opt/nomos/bin

# Nomos binaries.
COPY --from=bins /go/bin/git-importer git-importer
COPY --from=bins /go/bin/monitor monitor
COPY --from=bins /go/bin/nomos nomos

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

CMD echo "Please specify a specific binary:\n\n$(find . -type f -executable -printf '\t%f\n')\n" && exit 1
