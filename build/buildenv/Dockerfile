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

ARG GOLANG_CONTAINER=golang:1.14.2-alpine

# Environment to build the helper binaries from.
FROM ${GOLANG_CONTAINER} AS tools-base

# Adds support for building dynamically linked libraries.
RUN apk add --no-cache \
  bash \
  git

# Starting from go 1.10, build and test results are cached, which speeds up
# following runs.
ENV GOCACHE=/go/cache

ENV GO111MODULE=on
# Build static binaries
ENV CGO_ENABLED=0

# Install goimports.
# This is required by the `make fmt-go` target, even though it isn't used
# in building the Nomos image. This was previously included in golang:alpine,
# but not any more so we have to install it ourselves.
RUN go get golang.org/x/tools/cmd/goimports

# sembump knows how to increment semvers.
# sembump -kind=minor v1.2.3
# --> 1.3.0
# We can't just "go get" this dependency since "go get" throws an error if a
# git repository contains files which are case-insensitive collisions.
#
# We can't use go modules since this isn't really a go package.
ARG SEMBUMP_RELEASE="9ae1b9618feb2c76e88235410e9577e2b63d4576"
ARG SEMBUMP_REPO="github.com/jessfraz/junk"
WORKDIR /go/src/github.com/jessfraz
RUN git clone https://${SEMBUMP_REPO}.git
WORKDIR /go/src/${SEMBUMP_REPO}
RUN git checkout ${SEMBUMP_RELEASE}
RUN GO111MODULE=off go install ${SEMBUMP_REPO}/sembump

# semsort knows how to sort semvers.
# v1.0.2
# Example:
# echo "1.1.0 1.0.0" | semsort
# 1.0.0
# 1.1.0
RUN go get github.com/filmil/semsort/cmd/semsort@0ca60943363a67a752db9ed171e93a7e085eff0d

# gotopt2 parses command line options
# v0.1.2
ARG GOTOPT2_REPO="github.com/filmil/gotopt2"
RUN go get ${GOTOPT2_REPO}/cmd/gotopt2@v0.1.2

# The build environment docker file.
FROM ${GOLANG_CONTAINER}

# Adds support for building dynamically linked libraries.
RUN apk add --no-cache \
  bash \
  gcc \
  git \
  musl-dev \
  py-pip \
  upx

# grip is a Markdown rendered.
RUN pip install grip

# Starting from go 1.10, build and test results are cached, which speeds up
# following runs.
ENV GOCACHE=/go/cache

COPY --from=tools-base /go/bin/* /bin/
