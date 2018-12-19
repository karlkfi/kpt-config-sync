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

######################################################################

##### CONFIG #####

TOP_DIR := $(shell pwd)

# List of GOOS-GOARCH cross compilation targets
# Corresponding host mounts for Go std packages should be set below.
PLATFORMS ?= linux-amd64

REPO := github.com/google/nomos

# List of dirs containing go code owned by Nomos
NOMOS_CODE_DIRS := pkg cmd
NOMOS_GO_PKG := $(foreach dir,$(NOMOS_CODE_DIRS),./$(dir)/...)

# Directory containing all build artifacts.
OUTPUT_DIR := $(TOP_DIR)/.output

# Self-contained GOPATH dir.
GO_DIR := $(OUTPUT_DIR)/go

# Directory containing installed go binaries.
BIN_DIR := $(GO_DIR)/bin

# Directory used for staging Docker contexts.
STAGING_DIR := $(OUTPUT_DIR)/staging

# Directory used for staging the operator primitives.
OPERATOR_STAGING_DIR := ${OUTPUT_DIR}/staging/operator

# Directory used for staging docs.
DOCS_STAGING_DIR := $(STAGING_DIR)/docs

# Directory used for staging scripts.
SCRIPTS_STAGING_DIR := $(STAGING_DIR)/scripts

# Directory used for staging kubectl plugin release.
KUBECTL_PLUGIN_STAGING_DIR := $(STAGING_DIR)/kubectl-oidc

# Directory used for installer output.
INSTALLER_OUTPUT_DIR := $(STAGING_DIR)/installer_output

# Directory that gets mounted into /tmp for build and test containers.
TEMP_OUTPUT_DIR := $(OUTPUT_DIR)/tmp

# Directory containing gen-alld yaml files from manifest templates.
GEN_YAML_DIR := $(OUTPUT_DIR)/deployment

# Directory containing generated test yamls
TEST_GEN_YAML_DIR := $(OUTPUT_DIR)/test/yaml

# Directory containing templates yaml files.
TEMPLATES_DIR := $(TOP_DIR)/manifests/templates

# Directory containing test template yaml files.
TEST_TEMPLATES_DIR := $(TOP_DIR)/test/manifests/templates

# Use git tags to set version string.
VERSION := $(shell git describe --tags --always --dirty)

# UID of current user
UID := $(shell id -u)

# GID of current user
GID := $(shell id -g)

# GCP project that owns container registry for local dev build.
GCP_PROJECT ?= stolos-dev

# Prefix at which to store images in GCR. For using dev-local builds, the
# operator requires that the prefix of the image be unique to the build,
# and pulling of new versions is forced by setting a timestamp (DATE) in the
# image path.
GCR_PREFIX ?= $(GCP_PROJECT)/$(USER)/$(DATE)

# Docker image used for build and test. This image does not support CGO.
# NOTE: nomos-public is fully accessible publicly, do not use for anything
# other than buildenv
BUILDENV_PROJECT ?= nomos-public
BUILDENV_IMAGE_VERSION ?= v0.1.3
BUILDENV_IMAGE ?= gcr.io/$(BUILDENV_PROJECT)/buildenv:$(BUILDENV_IMAGE_VERSION)

# GCP service account credentials used in gcp e2e tests.
GCP_E2E_WATCHER_CRED := gs://stolos-dev/e2e/nomos-e2e.joonix.net/watcher_client_key.json
GCP_E2E_RUNNER_CRED := gs://stolos-dev/e2e/nomos-e2e.joonix.net/test_runner_client_key.json

# When set to "release", enables these optimizations:
# - Compress binary sizes
BUILD_MODE ?= debug

# All Nomos K8S deployments.
ALL_K8S_DEPLOYMENTS := syncer \
	git-policy-importer \
	gcp-policy-importer \
	monitor \
	resourcequota-admission-controller

# Nomos docker images containing all binaries.
NOMOS_IMAGE := nomos

# nomos binary for local run.
NOMOS_LOCAL := $(BIN_DIR)/linux_amd64/nomos

# Git server used in e2e tests.
GIT_SERVER_SRC := https://github.com/jkarlosb/git-server-docker.git

# Allows an interactive docker build or test session to be interrupted
# by Ctrl-C.  This must be turned off in case of non-interactive runs,
# like in CI/CD.
DOCKER_INTERACTIVE := --interactive
HAVE_TTY := $(shell [ -t 0 ] && echo 1 || echo 0)
ifeq ($(HAVE_TTY), 1)
	DOCKER_INTERACTIVE += --tty
endif

# Suppresses docker output on build.
#
# To turn off, for example:
#   make DOCKER_BUILD_QUIET="" deploy
DOCKER_BUILD_QUIET := --quiet

# Suppresses gcloud output.
GCLOUD_QUIET := --quiet

# Developer focused docker image tag.
DATE := $(shell date +'%s')
IMAGE_TAG ?= latest

DOCKER_RUN_ARGS := \
	$(DOCKER_INTERACTIVE)                                              \
	-e VERSION=$(VERSION)                                              \
	-e PKG=$(REPO)                                                     \
	-e BUILD_MODE=$(BUILD_MODE)                                        \
	-u $(UID):$(GID)                                                   \
	-v $(GO_DIR):/go                                                   \
	-v $$(pwd):/go/src/$(REPO)                                         \
	-v $(GO_DIR)/std/linux_amd64_static:/usr/local/go/pkg/linux_amd64_static    \
	-v $(GO_DIR)/std/darwin_amd64_static:/usr/local/go/pkg/darwin_amd64_static   \
	-v $(GO_DIR)/std/windows_amd64_static:/usr/local/go/pkg/windows_amd64_static   \
	-v $(TEMP_OUTPUT_DIR):/tmp                                         \
	-w /go/src/$(REPO)                                                 \
	--rm                                                               \
	$(BUILDENV_IMAGE)                                                  \

##### SETUP #####

.DEFAULT_GOAL := test-local

.PHONY: $(OUTPUT_DIR)
$(OUTPUT_DIR):
	@echo "+++ Creating the local build output directory: $(OUTPUT_DIR)"
	@mkdir -p \
		$(BIN_DIR) \
		$(GO_DIR)/pkg \
		$(GO_DIR)/src/$(REPO) \
	    $(GO_DIR)/std/linux_amd64_static \
	    $(GO_DIR)/std/darwin_amd64_static \
	    $(GO_DIR)/std/windows_amd64_static \
		$(INSTALLER_OUTPUT_DIR) \
		$(STAGING_DIR) \
		$(SCRIPTS_STAGING_DIR) \
		$(KUBECTL_PLUGIN_STAGING_DIR) \
		$(DOCS_STAGING_DIR) \
		$(GEN_YAML_DIR) \
		$(TEMP_OUTPUT_DIR) \
		$(TEST_GEN_YAML_DIR)

##### TARGETS #####

include Makefile.build
include Makefile.docs
include Makefile.e2e
include Makefile.installer
include Makefile.operator
include Makefile.bespin

# Redeploy a component without rerunning the installer.
redeploy-%: push-to-gcr-nomos gen-yaml-%
	@echo "+++ Redeploying without rerunning the installer: $*"
	@kubectl apply -f $(GEN_YAML_DIR)/$*.yaml

# Cleans all artifacts.
clean:
	@echo "+++ Cleaning $(OUTPUT_DIR)"
	@rm -rf $(OUTPUT_DIR)

test-unit: $(OUTPUT_DIR) pull-buildenv
	@echo "+++ Running unit tests in a docker container"
	@docker run $(DOCKER_RUN_ARGS) ./scripts/test-unit.sh $(NOMOS_GO_PKG)

# Runs unit tests and linter.
test: test-unit lint lint-bash

# Runs tests and local nomos vet tests.
test-local: test test-nomos-vet-local

# Runs all tests.
# This only runs on local dev environment not CI environment.
test-all-local: test test-nomos-vet-local test-e2e-all

goimports:
	@docker run $(DOCKER_RUN_ARGS) goimports -w $(NOMOS_CODE_DIRS)

lint: lint-go lint-bash lint-license

lint-go: build
	@docker run $(DOCKER_RUN_ARGS) ./scripts/lint.sh $(NOMOS_GO_PKG)

lint-bash:
	@./scripts/lint-bash.sh

lint-license: build
	@docker run $(DOCKER_RUN_ARGS) ./scripts/lint-license.sh

.PHONY: clientgen
clientgen:
	@echo "+++ Generating clientgen directory"
	$(TOP_DIR)/scripts/generate-clientset.sh
	$(TOP_DIR)/scripts/generate-watcher.sh

