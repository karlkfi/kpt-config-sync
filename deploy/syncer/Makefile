#!/usr/bin/make
#
# Copyright 2017 Kubernetes Authors
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
# Build rules for creating the syncer container.
#


GO ?= go
INSTALL ?= install
TEST ?= test

PACKAGE := syncer
REPO := github.com/google/stolos
PACKAGE_PATH := $(REPO)/cmd/$(PACKAGE)
BUILD_DIR := build

# label for docker image
IMAGE_REPOSITORY := $(PACKAGE)
IMAGE_TAG ?= test
IMAGE := $(IMAGE_REPOSITORY):$(IMAGE_TAG)

.PHONY: all build-go test-go build-docker
all: build-docker

test-go:
	$(GO) $(TEST) $(REPO)/...

build-docker: build-go Dockerfile
	docker build -t $(IMAGE) .

Dockerfile: tpl/Dockerfile
	m4 -DBUILD_DIR=$(BUILD_DIR) \
	   -DBINARY_NAME=$(PACKAGE) \
	   -DGEN_NOTE="Generated file, do not edit!" < $^ > $@

build-go: test-go
	$(GO) $(INSTALL) $(PACKAGE_PATH)
	mkdir -p $(BUILD_DIR)
	cp $(GO_BIN)/$(PACKAGE) $(BUILD_DIR)

clean:
	rm -rf build Dockerfile
	docker rmi $(IMAGE) || true

deploy:
	@echo "*** TODO(fmil): Implement target: $@"

undeploy:
	@echo "*** TODO(fmil): Implement target: $@"

push-docker:
	@echo "*** TODO(fmil): Implement target: $@"

