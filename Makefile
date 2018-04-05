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

# Directory used for staging docs.
DOCS_STAGING_DIR := $(STAGING_DIR)/docs

# Directory used for installer output.
INSTALLER_OUTPUT_DIR := $(STAGING_DIR)/installer_output

# Directory containing gen-alld yaml files from manifest templates.
GEN_YAML_DIR := $(OUTPUT_DIR)/yaml

# Directory containing templates yaml files.
TEMPLATES_DIR := $(TOP_DIR)/manifests/enrolled/templates

# Use git tags to set version string.
VERSION := $(shell git describe --tags --always --dirty)

# Which architecture to build.
ARCH ?= amd64

# Docker image used for build and test. This image does not support CGO.
BUILD_IMAGE ?= buildenv

# GCP project that owns container registry for local dev build.
GCP_PROJECT ?= stolos-dev

# All Nomos apps which run on K8S.
ALL_K8S_APPS := syncer \
	git-policy-importer \
	resourcequota-admission-controller \
	policynodes-admission-controller

# ALl Nomos apps
ALL_APPS := $(ALL_K8S_APPS) \
	nomosvet

# Allows an interactive docker build or test session to be interrupted
# by Ctrl-C.  This must be turned off in case of non-interactive runs,
# like in CI/CD.
DOCKER_INTERACTIVE := --interactive
HAVE_TTY := $(shell [ -t 0 ] && echo 1 || echo 0)
ifeq ($(HAVE_TTY), 1)
	DOCKER_INTERACTIVE += --tty
endif

# Developer focused docker image tag.
DATE := $(shell date +'%s')
IMAGE_TAG ?= $(VERSION)-$(USER)-$(DATE)

##### SETUP #####

.DEFAULT_GOAL := test

# Creates the local golang build output directory.
.output:
	mkdir -p \
		$(BIN_DIR)/$(ARCH) \
		$(GEN_YAML_DIR) \
		$(GO_DIR)/pkg \
		$(GO_DIR)/src/$(REPO) \
		$(GO_DIR)/std/$(ARCH) \
		$(INSTALLER_OUTPUT_DIR) \
		$(STAGING_DIR)

##### TARGETS #####

# Uses a custom build environment.  See build/buildenv/Dockerfile for the
# details on the build environment.
.PHONY: buildenv
buildenv:
	@docker build build/buildenv --tag=$(BUILD_IMAGE)

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
image-all: $(addprefix image-, $(ALL_APPS))
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
push-to-gcr-all: $(addprefix push-to-gcr-, $(ALL_APPS))
	@echo "finished pushing to all"

# Pushes the specified component's docker image ot gcr.io.
push-to-gcr-%: image-%
	gcloud docker -- push gcr.io/$(GCP_PROJECT)/$*:$(IMAGE_TAG)

# Creates staging directory for building installer docker image.
installer-staging: push-to-gcr-all gen-yaml-all
	cp -r $(TOP_DIR)/build/installer $(STAGING_DIR)
	cp $(BIN_DIR)/$(ARCH)/installer $(STAGING_DIR)/installer
	cp $(BIN_DIR)/$(ARCH)/configgen $(STAGING_DIR)/installer
	mkdir -p $(STAGING_DIR)/installer/yaml
	cp $(OUTPUT_DIR)/yaml/*  $(STAGING_DIR)/installer/yaml
	mkdir -p $(STAGING_DIR)/installer/manifests/enrolled
	cp -r $(TOP_DIR)/manifests/enrolled/* $(STAGING_DIR)/installer/manifests/enrolled
	mkdir -p $(STAGING_DIR)/installer/manifests/common
	cp -r $(TOP_DIR)/manifests/common/* $(STAGING_DIR)/installer/manifests/common
	mkdir -p $(STAGING_DIR)/installer/examples
	cp -r $(TOP_DIR)/build/installer/examples/* $(STAGING_DIR)/installer/examples
	cp $(TOP_DIR)/build/installer/entrypoint.sh $(STAGING_DIR)/installer
	mkdir -p $(STAGING_DIR)/installer/scripts
	cp $(TOP_DIR)/scripts/deploy-resourcequota-admission-controller.sh \
		$(STAGING_DIR)/installer/scripts
	cp $(TOP_DIR)/scripts/generate-resourcequota-admission-controller-certs.sh \
		$(STAGING_DIR)/installer/scripts
	cp $(TOP_DIR)/scripts/deploy-policynodes-admission-controller.sh \
		$(STAGING_DIR)/installer/scripts
	cp $(TOP_DIR)/scripts/generate-policynodes-admission-controller-certs.sh \
		$(STAGING_DIR)/installer/scripts


# Builds the installer docker image using the nomos release in .output
installer-image: installer-staging
	docker build -t gcr.io/$(GCP_PROJECT)/installer:$(IMAGE_TAG) \
		--build-arg "INSTALLER_VERSION=$(IMAGE_TAG)" \
		$(STAGING_DIR)/installer
	gcloud docker -- push gcr.io/$(GCP_PROJECT)/installer:$(IMAGE_TAG)

# Runs the installer via docker in interactive mode.
deploy-interactive: installer-image
	@./scripts/run-installer.sh \
		--interactive \
		--container=gcr.io/$(GCP_PROJECT)/installer \
		--output_dir=$(INSTALLER_OUTPUT_DIR) \
		--version=$(IMAGE_TAG)

# Runs the installer via docker in batch mode using the installer config
# file specied in the environment variable NOMOS_INSTALLER_CONFIG.
deploy: installer-image
	@if [ -z "$(NOMOS_INSTALLER_CONFIG)" ]; then \
		echo 'must set NOMOS_INSTALLER_CONFIG to use make deploy'; \
		exit 1; \
	fi; \
	./scripts/run-installer.sh \
		--config=$(NOMOS_INSTALLER_CONFIG) \
		--container=gcr.io/$(GCP_PROJECT)/installer \
		--output_dir=$(INSTALLER_OUTPUT_DIR) \
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
	e2e/e2e.sh --logtostderr --vmodule=bash=10,exec=10,kubectl=10
# Runs end-to-end tests but leaves files for debugging.
test-e2e-no-cleanup:
	e2e/e2e.sh -skip_cleanup --logtostderr --vmodule=bash=10,exec=10,kubectl=10

# Runs all tests.
test-all: test test-e2e

# Generate K8S client-set.
gen-client-set:
	$(TOP_DIR)/scripts/generate-clientset.sh

# Installs Nomos kubectl plugin.
install-kubectl-plugin:
	cd $(TOP_DIR)/cmd/kubectl-nomos; ./install.sh

# Generates the podspec yaml for each component.
gen-yaml-all: $(addprefix gen-yaml-, $(ALL_K8S_APPS))
	@echo "finished generating all yaml."

# Generates the podspec yaml for the component specified.
gen-yaml-%:
	m4 -DIMAGE_NAME=gcr.io/$(GCP_PROJECT)/$*:$(IMAGE_TAG) < \
			$(TEMPLATES_DIR)/$*.yaml > $(GEN_YAML_DIR)/$*.yaml

# Creates staging directory for generating docs.
docs-staging: .output
	rm -rf $(DOCS_STAGING_DIR)/*.html $(DOCS_STAGING_DIR)/img
	cp -r $(TOP_DIR)/docs $(STAGING_DIR)
	cp $(TOP_DIR)/README.md $(DOCS_STAGING_DIR)

# Checks that grip binary is installed.
check-for-grip:
	@command -v grip || (echo "Need to install grip: pip install grip" && exit 1)

# Converts Markdown docs into HTML.
docs-generate: buildenv docs-staging
	@docker run $(DOCKER_INTERACTIVE) \
		-u $$(id -u):$$(id -g)                                             \
		-v $$(pwd):/go/src/$(REPO)                                         \
		-w /go/src/$(REPO)                                                 \
		-e HOME=/go/src/$(REPO)/.output/config                             \
		-e REPO=$(REPO)                                                    \
		-e DOCS_STAGING_DIR=/go/src/$(REPO)/.output/staging/docs           \
		-e TOP_DIR=/go/src/$(REPO)                                         \
		--rm                                                               \
		$(BUILD_IMAGE)                                                     \
		/bin/bash -c " \
		    ./scripts/docs-generate.sh \
	    "
	@echo "Open in your browser: file://$(DOCS_STAGING_DIR)/README.html"

# Packages HTML docs into a zip package.
docs-package: docs-generate
	cd $(STAGING_DIR); rm -f nomos-docs.zip; zip -r nomos-docs.zip docs
