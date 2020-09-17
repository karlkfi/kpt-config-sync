# Use cached build of reconciler-manager, go-binaries should have been built just previously
FROM gcr.io/nomos-dev/go-binaries:latest as bins

RUN stat /go/bin/reconciler-manager

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=bins /go/bin/reconciler-manager manager
USER nonroot:nonroot

# License file required for on-prem release.
COPY LICENSE LICENSE

# Copy in golang-lru source (MPL licensed dependency)
COPY vendor/github.com/hashicorp/golang-lru/ golang-lru/

ENTRYPOINT ["/manager"]
