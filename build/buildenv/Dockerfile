#!/usr/bin/make
#
# Copyright 2018 CSP Config Management Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

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

# The build environment docker file.
FROM golang:1.11.2-alpine

# Adds support for building dynamically linked libraries.
RUN apk add --no-cache \
  bash \
  gcc \
  musl-dev \
  py-pip \
  git \
  upx

RUN pip install grip

# Starting from go 1.10, build and test results are cached, which speeds up
# following runs.
ENV GOCACHE=/go/cache

RUN go get gopkg.in/alecthomas/gometalinter.v2 && gometalinter.v2 --install

# sembump knows how to increment semvers.
# sembump -kind=minor v1.2.3
# --> 1.3.0
RUN go get github.com/jessfraz/junk/sembump && \
    go install github.com/jessfraz/junk/sembump

# semsort knows how to sort semvers.
# echo "1.1.0 1.0.0" | semsort
# 1.0.0
# 1.1.0
RUN go get github.com/filmil/semsort/... && \
    go install github.com/filmil/semsort/...

RUN mv /go/bin/* /bin
