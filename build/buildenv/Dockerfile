#
# NOTE: When updating this file, you will also need to do the following:
#
# 1. Update the BUILDENV_IMAGE_VERSION variable in Makefile
# 2. Run the following locally to test the build then push the image so it's
#    accessible by CI:
#    $ make build-buildenv && make build && make push-buildenv
# 3. Upload your change for review and check the CI results.  Note that Prow
#    will fail if you have not uploaded your new image.
#

ARG GOLANG_CONTAINER=golang:1.11.2-alpine

# Environment to build the helper binaries from.
FROM ${GOLANG_CONTAINER} AS tools-base

# Adds support for building dynamically linked libraries.
RUN apk add --no-cache \
  bash \
  git

# Starting from go 1.10, build and test results are cached, which speeds up
# following runs.
ENV GOCACHE=/go/cache
# Build static binaries
ENV CGO_ENABLED=0

# gometalinter is a program that runs all go linters.
ARG GOMETALINTERV2_RELEASE="102ac984005d45456a7e3ae6dc94ebcd95c2bb19"
ARG GOMETALINTERV2_REPO="gopkg.in/alecthomas/gometalinter.v2"
RUN go get ${GOMETALINTERV2_REPO}
WORKDIR /go/src/${GOMETALINTERV2_REPO}
RUN git checkout ${GOMETALINTERV2_RELEASE} && \
    gometalinter.v2 --install

# sembump knows how to increment semvers.
# sembump -kind=minor v1.2.3
# --> 1.3.0
ARG SEMBUMP_RELEASE="9ae1b9618feb2c76e88235410e9577e2b63d4576"
ARG SEMBUMP_REPO="github.com/jessfraz/junk"
RUN go get ${SEMBUMP_REPO}/sembump
WORKDIR /go/src/${SEMBUMP_REPO}
RUN git checkout ${SEMBUMP_RELEASE} && go install ${SEMBUMP_REPO}/sembump

# semsort knows how to sort semvers.
# v1.0.2
# Example:
# echo "1.1.0 1.0.0" | semsort
# 1.0.0
# 1.1.0
ARG SEMSORT_RELEASE="0ca60943363a67a752db9ed171e93a7e085eff0d"
ARG SEMSORT_REPO="github.com/filmil/semsort"
RUN go get ${SEMSORT_REPO}/...
WORKDIR /go/src/${SEMSORT_REPO}
RUN git checkout ${SEMSORT_RELEASE} && go install ${SEMSORT_REPO}/...

# gotopt2 knows how to parse command line options
ARG GOTOPT2_RELEASE="v0.1.2"
ARG GOTOPT2_REPO="github.com/filmil/gotopt2"
RUN go get ${GOTOPT2_REPO}/...
WORKDIR /go/src/${GOTOPT2_REPO}
RUN git checkout ${GOTOPT2_RELEASE} && go install ${GOTOPT2_REPO}/...

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
