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

REPO := github.com/google/nomos

# List of dirs containing go code owned by Nomos
NOMOS_CODE_DIRS := pkg cmd

# Directory containing all build artifacts.
OUTPUT_DIR := $(TOP_DIR)/.output

# Self-contained GOPATH dir.
GO_DIR := $(OUTPUT_DIR)/go

# Directory containing installed go binaries.
BIN_DIR := $(GO_DIR)/bin

# Directory used for staging Docker contexts.
STAGING_DIR := $(OUTPUT_DIR)/staging

# Directory containing gen-alld yaml files from manifest templates.
GEN_YAML_DIR := $(OUTPUT_DIR)/yaml

# Directory containing templates yaml files.
TEMPLATES_DIR := $(TOP_DIR)/manifests/enrolled/templates

# Use git tags to set version string.
VERSION := $(shell git describe --tags --always --dirty)

# Which architecture to build.
ARCH ?= amd64

# Docker image used for build and test.  This image does not support CGO.
BUILD_IMAGE ?= buildenv

# GCP project that owns container registry.
GCP_PROJECT ?= stolos-dev

# Whether this is an official release or dev workflow.
RELEASE ?= 0

# Is the current release considered "stable".  Set to 0 for e.g. release
# candidates.
STABLE ?= 0

# The GCS bucket to release into for 'make release'
RELEASE_BUCKET := gs://$(GCP_PROJECT)

# All Nomos components
ALL_COMPONENTS := syncer \
	git-policy-importer \
	resourcequota-admission-controller

# Allows an interactive docker build or test session to be interrupted
# by Ctrl-C.  This must be turned off in case of non-interactive runs,
# like in CI/CD.
DOCKER_INTERACTIVE ?= --interactive --tty

# Docker image tag.
ifeq ($(RELEASE), 1)
	IMAGE_TAG := $(VERSION)
else
	IMAGE_TAG := $(VERSION)-$(shell date +'%s')
endif

##### SETUP #####

# set build to be default
.DEFAULT_GOAL := test

# Creates the local golang build output directory.
.output:
	mkdir -p \
		$(GO_DIR)/src/$(REPO) \
		$(GO_DIR)/pkg \
		$(GO_DIR)/std/$(ARCH) \
		$(BIN_DIR)/$(ARCH) \
		$(STAGING_DIR) \
		$(GEN_YAML_DIR)

##### TARGETS #####

# Uses a custom build environment.  See toolkit/buildenv/Dockerfile for the
# details on the build environment.
.PHONY: buildenv
buildenv:
	@docker build toolkit/buildenv --tag=$(BUILD_IMAGE)

# Runs all static analyzers and autocorrects.
lint:
	goimports -w $(NOMOS_CODE_DIRS)

# Compiles nomos hermetically using a docker container.
.PHONY: build
build: buildenv .output
	@docker run $(DOCKER_INTERACTIVE)                                      \
		-u $$(id -u):$$(id -g)                                             \
		-v $(GO_DIR):/go                                                   \
		-v $$(pwd):/go/src/$(REPO)                                         \
		-v $(BIN_DIR)/$(ARCH):/go/bin                                      \
		-v $(GO_DIR)/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static    \
		-w /go/src/$(REPO)                                                 \
		--rm                                                               \
		$(BUILD_IMAGE)                                                     \
		/bin/sh -c "                                                       \
			ARCH=$(ARCH)                                                   \
			VERSION=$(VERSION)                                             \
			PKG=$(REPO)                                                    \
			./scripts/build/build.sh                                       \
		"
# Creates a docker image for each nomos component.
image-all: $(addprefix image-, $(ALL_COMPONENTS))
	@echo "finished packaging all"

# Creates a docker image for the specified nomos component.
image-%: build
	@echo
	@echo "******* Building image for $* *******"
	@echo
	cp -r $(TOP_DIR)/build/$* $(STAGING_DIR)
	cp $(BIN_DIR)/$(ARCH)/$* $(STAGING_DIR)/$*
	docker build -t gcr.io/$(GCP_PROJECT)/$*:$(IMAGE_TAG) $(STAGING_DIR)/$*

# Pushes each component's docker image to gcr.io.
push-to-gcr-all: $(addprefix push-to-gcr-, $(ALL_COMPONENTS))
	@echo "finished pushing to all"

# Pushes the specified component's docker image ot gcr.io.
push-to-gcr-%: image-%
	gcloud docker -- push gcr.io/$(GCP_PROJECT)/$*:$(IMAGE_TAG)

# Builds the installer using the nomos release in .output
installer: GCP_PROJECT=stolos-dev
installer: push-to-gcr-all gen-yaml-all
	ARCH=$(ARCH) \
	BIN_DIR=$(BIN_DIR) \
	GCP_PROJECT=$(GCP_PROJECT) \
	IMAGE_TAG=$(IMAGE_TAG) \
	OUTPUT_DIR=$(OUTPUT_DIR) \
	RELEASE_BUCKET=$(RELEASE_BUCKET) \
	STAGING_DIR=$(STAGING_DIR) \
	TOP_DIR=$(TOP_DIR) \
	VERSION=$(VERSION) \
	STABLE=$(STABLE) \
		$(TOP_DIR)/scripts/push-installer.sh

# Runs the installer via docker in interactive mode.
deploy-interactive: installer
	@./scripts/run-installer.sh \
		--interactive \
		--container=gcr.io/$(GCP_PROJECT)/installer \
		--version=$(IMAGE_TAG)

# Runs the installer via docker in batch mode using the installer config
# file specied in the environment variable NOMOS_INSTALLER_CONFIG.
deploy: installer
	@if [ -z "$(NOMOS_INSTALLER_CONFIG)" ]; then \
		echo 'must set NOMOS_INSTALLER_CONFIG to use make deploy'; \
		exit 1; \
	fi; \
	./scripts/run-installer.sh \
		--config=$(NOMOS_INSTALLER_CONFIG) \
		--container=gcr.io/$(GCP_PROJECT)/installer \
		--version=$(IMAGE_TAG)

# Cleans all artifacts.
clean:
	rm -rf $(OUTPUT_DIR)

# Runs unit tests in a docker container.
test: buildenv .output
	@docker run $(DOCKER_INTERACTIVE)                                      \
		-u $$(id -u):$$(id -g)                                             \
		-v $(GO_DIR):/go                                                   \
		-v $$(pwd):/go/src/$(REPO)                                         \
		-v $(BIN_DIR)/$(ARCH):/go/bin                                      \
		-v $(GO_DIR)/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static    \
		-w /go/src/$(REPO)                                                 \
		--rm                                                               \
		$(BUILD_IMAGE)                                                     \
		/bin/sh -c "                                                       \
			ARCH=$(ARCH)                                                   \
			VERSION=$(VERSION)                                             \
			PKG=$(REPO)                                                    \
			./scripts/build/test.sh $(NOMOS_CODE_DIRS)                    \
		"

# Runs end-to-end tests.
test-e2e:
	e2e/e2e.sh
# Runs end-to-end tests but leaves files for debugging.
test-e2e-no-cleanup:
	e2e/e2e.sh -skip_cleanup

# Generate K8S client-set.
gen-client-set:
	$(TOP_DIR)/scripts/generate-clientset.sh

# Installs Nomos kubectl plugin.
install-kubectl-plugin:
	cd $(TOP_DIR)/cmd/kubectl-nomos; ./install.sh

# Generates the podspec yaml for each component.
gen-yaml-all: $(addprefix gen-yaml-, $(ALL_COMPONENTS))
	@echo "finished generating all yaml."

# Generates the podspec yaml for the component specified.
gen-yaml-%:
	m4 -DIMAGE_NAME=gcr.io/$(GCP_PROJECT)/$*:$(IMAGE_TAG) < \
			$(TEMPLATES_DIR)/$*.yaml > $(GEN_YAML_DIR)/$*.yaml
