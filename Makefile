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

######################################################################
# This part of the configuration is editable.
# Configuration bits.
TOPDIR := $(shell pwd)

# Set this environment variable to set where the end result is to be deployed.
export DEPLOYMENT_TYPE ?= gce

# This is the default top-level repo path.
export REPO ?= github.com/google/stolos

# The binaries compiled for Docker will go here.  Binaries that are not
# targeted for docker (e.g. for local testing) will be output in the
# "regular" system $GOBIN.
export GO_BIN := $(TOPDIR)/.build-out/bin

# Labels for docker images.
export IMAGE_TAG ?= test

# This is the list of binaries to build.
export BINARIES := admission-controller authorizer syncer

# List of dirs containing go code owned by Stolos
export STOLOS_CODE_DIRS := pkg cmd

# The GCP project that will be used to deploy all artifacts.
export GCP_PROJECT := $(shell gcloud config get-value project)
export GCP_ZONE := us-central1-b

# The Kubernetes version currently used for deployment.
export KUBERNETES_VERSION := 1.8.0-stolos

######################################################################
# Do not edit beyond this point.

define HELP_BODY
Stolos Build and Deploy
========================
Some targets need specific environment variables set.  See below for
requirements.  (for target 'foo' the command is 'make foo')

help:
	Prints this text.
test:
	Tests everything.
build:
	Builds everything.
build-static-go:
	Builds all Stolos binaries as stand-alone statically linked binaries.
	Output is stored in ./.build-out/...
build-docker:
	REQUIRES: $$DEPLOYMENT_TYPE
	Builds all docker containers.
build-docker-%:
	Builds only a specific binary.
	Example: build-docker-syncer
push-docker:
	REQUIRES: $$DEPLOYMENT_TYPE
	Pushes all docker containers.
deploy:
	REQUIRES: $$DEPLOYMENT_TYPE
	Deploys all binaries.
deploy-%:
	REQUIRES: $$DEPLOYMENT_TYPE
	Deploys a specific binary.
undeploy:
	REQUIRES: $$DEPLOYMENT_TYPE
	Undeploys all binaries.
undeploy-%:
	REQUIRES: $$DEPLOYMENT_TYPE
	Undeploys a specific binary.
clean:
	Clean everything up
lint:
	Runs all static analyzers such as gofmt and go vet.

Environment Variables
----------------------
Alternatives are {a|b}
The first alternative is the default.

DEPLOYMENT_TYPE: {gce|minikube|local}
	Used to choose system-specific deployment where it applies.
	  - gce: deploy to GCE.  Requires gcloud setup.
      - minikube: deploy to minikube.  Requries minikube up and running.
	  - local: deploy to a locally running docker.  Please note, only some
	  targets support this deployment, e.g. build-docker-local, and
	  push-docker-local.  Requires docker to be running on your local machine.

GCP_PROJECT:
	The GCP project name used to push containers or to deploy.

endef

export HELP_BODY
.PHONY: help
help:
	@echo "$$HELP_BODY"

# Test everything.
.PHONY: test
test: test-go

.PHONY: test-go
test-go:
	go test $(REPO)/...

# Runs all tests on the statically built go binaries.
.PHONY: test-static-go
test-static-go: .build-out
	CGO_ENABLED=0
	go test -i -installsuffix="static" $(REPO)/...
	go test -installsuffix="static" $(REPO)/...

# Builds everything.
.PHONY: build
build: build-go

.PHONY: build-go
build-go:
	go build -i $(REPO)/...

# Creates the local golang build output directory.
.build-out:
	mkdir -p $(GO_BIN)/{bin,pkg,src}

# Build statically linked binaries.  These are good for deploying into very
# small containers, e.g. built off of "scratch" or "busybox" or some such.
.PHONY: build-go-static
build-go-static: .build-out
	CGO_ENABLED=0 GOBIN=$(GO_BIN) \
	      go install -installsuffix="static" $(REPO)/cmd/...

### make build-docker ...

.PHONY: build-docker
build-docker: build-go-static
	make -C deploy build-docker

.PHONY: build-docker
build-docker-%: build-go-static
	echo "Building: $@"
	make -e -C deploy $@

### make push-docker ...

.PHONY: push-docker
push-docker: build-go-static
	make -C deploy push-docker


push-docker-%: build-go-static
	make -e -C deploy $@

### make deploy ...
#
# Uses DEPLOYMENT_TYPE=...

.PHONY: deploy
deploy: build-go-static
	make -C deploy deploy

deploy-%: build-go-static
	make -e -C deploy $@

### make undeploy ...
#
# Uses DEPLOYMENT_TYPE=...

.PHONY: undeploy
undeploy:
	make -C deploy undeploy

undeploy-%:
	make -e -C deploy $@

### make clean ...

.PHONY: clean
clean:
	go clean $(REPO)
	rm -fr .build-out
	make -C deploy clean

clean-%:
	make -C deploy $@

# Run all static analyzers.
.PHONY: lint
lint:
	gofmt -w $(STOLOS_CODE_DIRS)
	go tool vet $(STOLOS_CODE_DIRS)
