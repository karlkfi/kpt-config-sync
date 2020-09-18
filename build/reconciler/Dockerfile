# Use cached build of reconciler, go-binaries should have been built just previously
FROM gcr.io/nomos-dev/go-binaries:latest as bins

RUN stat /go/bin/reconciler

FROM k8s.gcr.io/debian-base:v2.0.0
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
