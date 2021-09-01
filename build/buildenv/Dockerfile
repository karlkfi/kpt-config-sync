#
# NOTE: When updating this file, you will also need to do the following:
#
# 1. Increment the BUILDENV_IMAGE_VERSION variable in Makefile
# 2. Run the following locally to test the build then push the image so it's
#    accessible by CI:
#    $ make build-buildenv && make build && make push-buildenv
# 3. Upload your change for review and check the CI results.  Note that Prow
#    will fail if you have not uploaded your new image.
#

ARG GOLANG_CONTAINER=gcr.io/gae-runtimes/go116_app_builder:go116_20210829_1_16_7_RC00

# Environment to build the helper binaries from.
FROM ${GOLANG_CONTAINER} AS tools-base

# Set GOPATH since it is not already set.
ENV GOPATH=/go

# Update PATH with go specific locations.
ENV PATH "/go/bin:/usr/local/go/bin:$PATH"

RUN apt-get update \
  && apt-get install -y \
  bash \
  git

ENV GO111MODULE=on
# Build static binaries
ENV CGO_ENABLED=0

# Install goimports.
# This is required by the `make fmt-go` target, even though it isn't used
# in building the Nomos image. This was previously included in golang:alpine,
# but not any more so we have to install it ourselves.
RUN go get golang.org/x/tools/cmd/goimports

# gotopt2 parses command line options
# v0.1.2
ARG GOTOPT2_REPO="github.com/filmil/gotopt2"
RUN go get ${GOTOPT2_REPO}/cmd/gotopt2@v0.1.2

# The build environment docker file.
FROM ${GOLANG_CONTAINER}

# Set GOPATH since it is not already set.
ENV GOPATH=/go

# Update PATH with go specific locations.
ENV PATH "/go/bin:/usr/local/go/bin:$PATH"

# Adds support for building dynamically linked libraries.
RUN apt-get update \
  && apt-get install -y \
  bash \
  gcc \
  git \
  musl-dev \
  python-pip \
  upx \
  wget

# Starting from go 1.10, build and test results are cached, which speeds up
# following runs.
ENV GOCACHE=/go/cache

COPY --from=tools-base /go/bin/* /bin/

# To disable an existing ENTRYPOINT, set an empty array. This allows arguments
# to 'docker run' to exec as normal.
ENTRYPOINT []
