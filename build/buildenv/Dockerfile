#!/usr/bin/make
#
# Copyright 2018 Nomos Authors
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
