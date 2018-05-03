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
NOMOS_GO_PKG := $(foreach dir,$(NOMOS_CODE_DIRS),./$(dir)/...)

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

# UID of current user
UID := $(shell id -u)

# GID of current user
GID := $(shell id -g)

# Docker image used for build and test. This image does not support CGO.
BUILD_IMAGE ?= buildenv

# GCP project that owns container registry for local dev build.
GCP_PROJECT ?= stolos-dev

# All Nomos apps which run on K8S.
ALL_K8S_APPS := syncer \
	git-policy-importer \
	resourcequota-admission-controller \
	policynodes-admission-controller

# ALl Nomos apps that are dockerized.
ALL_APPS := $(ALL_K8S_APPS) \
	nomosvet

GIT_SERVER_SRC := https://github.com/jkarlosb/git-server-docker.git

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

DOCKER_RUN_ARGS := \
	$(DOCKER_INTERACTIVE)                                              \
	-e ARCH=$(ARCH)                                                    \
	-e VERSION=$(VERSION)                                              \
	-e PKG=$(REPO)                                                     \
	-u $$(id -u):$$(id -g)                                             \
	-v $(GO_DIR):/go                                                   \
	-v $$(pwd):/go/src/$(REPO)                                         \
	-v $(BIN_DIR)/$(ARCH):/go/bin                                      \
	-v $(GO_DIR)/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static    \
	-w /go/src/$(REPO)                                                 \
	--rm                                                               \
	$(BUILD_IMAGE)                                                     \

##### SETUP #####

.DEFAULT_GOAL := test

# Creates the local golang build output directory.
$(OUTPUT_DIR):
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
$(OUTPUT_DIR)/buildenv: build/buildenv/Dockerfile $(OUTPUT_DIR)
	@docker build $(dir $<) --tag=$(BUILD_IMAGE)
	@echo "This file exists to track that buildenv is up to date" > $@

buildenv: $(OUTPUT_DIR)/buildenv

# Compiles nomos hermetically using a docker container.
.PHONY: build
build: buildenv
	@docker run $(DOCKER_RUN_ARGS) ./scripts/build.sh

# Creates a docker image for each nomos component.
image-all: $(addprefix image-, $(ALL_APPS))
	@echo "Finished packaging all"

# Creates a docker image for the specified nomos component.
image-%: build
	@echo
	@echo "******* Building image for $* *******"
	@echo
	cp -r $(TOP_DIR)/build/$* $(STAGING_DIR)
	cp $(BIN_DIR)/$(ARCH)/$* $(STAGING_DIR)/$*
	docker build -t gcr.io/$(GCP_PROJECT)/$*:$(IMAGE_TAG) $(STAGING_DIR)/$*

# Creates docker image for the test git-server from github source
image-git-server:
	@echo
	@echo "******* Building image for test git server *******"
	@echo
	docker build -t gcr.io/$(GCP_PROJECT)/git-server:$(IMAGE_TAG) $(GIT_SERVER_SRC)

# Pushes each component's docker image to gcr.io.
push-to-gcr-all: $(addprefix push-to-gcr-, $(ALL_APPS))
	@echo "finished pushing to all"

# Pushes the specified component's docker image ot gcr.io.
push-to-gcr-%: image-%
	gcloud docker -- push gcr.io/$(GCP_PROJECT)/$*:$(IMAGE_TAG)

# Generates the podspec yaml for each component.
gen-yaml-all: $(addprefix gen-yaml-, $(ALL_K8S_APPS))
	@echo "Finished generating all yaml."

# Generates the podspec yaml for the component specified.
gen-yaml-%:
	m4 -DIMAGE_NAME=gcr.io/$(GCP_PROJECT)/$*:$(IMAGE_TAG) < \
			$(TEMPLATES_DIR)/$*.yaml > $(GEN_YAML_DIR)/$*.yaml

# Creates staging directory for building installer docker image.
installer-staging: push-to-gcr-all gen-yaml-all
	cp -r $(TOP_DIR)/build/installer $(STAGING_DIR)
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

# Builds the installer docker image using the nomos release in $(OUTPUT_DIR)
installer-image: installer-staging
	docker build -t gcr.io/$(GCP_PROJECT)/installer:$(IMAGE_TAG) \
		--build-arg "INSTALLER_VERSION=$(IMAGE_TAG)" \
		$(STAGING_DIR)/installer
	gcloud docker -- push gcr.io/$(GCP_PROJECT)/installer:$(IMAGE_TAG)

# Runs the installer via docker in interactive mode.
deploy-interactive: installer-image
	$(TOP_DIR)/scripts/run-installer.sh \
		--interactive \
		--container=gcr.io/$(GCP_PROJECT)/installer \
		--output_dir=$(INSTALLER_OUTPUT_DIR) \
		--version=$(IMAGE_TAG)

deploy-test-git-server: $(OUTPUT_DIR) image-git-server push-to-gcr-git-server \
	gen-yaml-git-server

deploy-test-e2e: $(OUTPUT_DIR) deploy-test-git-server deploy

# Runs the installer via docker in batch mode using the installer config
# file specied in the environment variable NOMOS_INSTALLER_CONFIG.
deploy: installer-image
	@if [ -z "$(NOMOS_INSTALLER_CONFIG)" ]; then \
		echo 'Must set NOMOS_INSTALLER_CONFIG to use make deploy'; \
		exit 1; \
	fi; \
	./scripts/run-installer.sh \
		--config=$(NOMOS_INSTALLER_CONFIG) \
		--container=gcr.io/$(GCP_PROJECT)/installer \
		--output_dir=$(INSTALLER_OUTPUT_DIR) \
		--use_current_context=$(USE_CURRENT_CONTEXT) \
		--version=$(IMAGE_TAG)

# Creates staging directory for building e2e docker image. Note that it is basically a copy of the
# staging directory for the installer plus a few extras for e2e.
e2e-staging: installer-image
	mkdir -p $(STAGING_DIR)/e2e-tests
	cp -r $(STAGING_DIR)/installer/* $(STAGING_DIR)/e2e-tests
	cp -r $(TOP_DIR)/build/e2e-tests/* $(STAGING_DIR)/e2e-tests
	cp -r $(TOP_DIR)/e2e $(OUTPUT_DIR)
	cp -r $(TOP_DIR)/examples/acme/sot $(OUTPUT_DIR)/e2e
	cp -n $(HOME)/.ssh/id_rsa.nomos $(OUTPUT_DIR)/e2e/id_rsa.nomos
	cp $(HOME)/.ssh/id_rsa.nomos.pub $(OUTPUT_DIR)/e2e/id_rsa.nomos.pub
	cp $(TOP_DIR)/scripts/init-git-server.sh $(OUTPUT_DIR)/e2e/
	cp $(GEN_YAML_DIR)/git-server.yaml $(OUTPUT_DIR)/e2e/

# Builds the e2e docker image. Note that the GCP project is hardcoded since we currently don't want
# or need a nomos-release version of the e2e-tests image.
e2e-image: deploy-test-git-server e2e-staging
	docker build -t gcr.io/stolos-dev/e2e-tests:$(IMAGE_TAG) \
		--build-arg "VERSION=$(IMAGE_TAG)" \
		--build-arg "UID=$(UID)" \
		--build-arg "GID=$(GID)" \
		--build-arg "UNAME=$(USER)" \
		--build-arg "GCP_PROJECT=$(GCP_PROJECT)" \
		$(STAGING_DIR)/e2e-tests
	gcloud docker -- push gcr.io/stolos-dev/e2e-tests:$(IMAGE_TAG)

# Runs end-to-end tests.
test-e2e: clean e2e-image
	mkdir -p ${INSTALLER_OUTPUT_DIR}/{kubeconfig,certs,gen_configs,logs}
	@docker run -it \
	    -u $$(id -u):$$(id -g) \
	    -v "${HOME}":/home/user \
	    -v "${INSTALLER_OUTPUT_DIR}/certs":/opt/installer/certs \
	    -v "${INSTALLER_OUTPUT_DIR}/gen_configs":/opt/installer/gen_configs \
	    -v "${INSTALLER_OUTPUT_DIR}/kubeconfig":/opt/installer/kubeconfig \
	    -v "${INSTALLER_OUTPUT_DIR}/logs":/tmp \
	    -v "$(TOP_DIR)/examples":/opt/installer/configs \
	    -v "$(OUTPUT_DIR)/e2e":/opt/testing \
	    -e "VERSION=${IMAGE_TAG}" \
	    "gcr.io/stolos-dev/e2e-tests:${IMAGE_TAG}" "$@" \
	    && echo "+++ E2E tests completed." \
	    || (echo "### E2E tests failed. Logs are available in ${INSTALLER_OUTPUT_DIR}/logs"; exit 1)

# Redeploys all components to cluster without rerunning the installer.
redeploy-all: $(addprefix redeploy-, $(ALL_K8S_APPS))
	@echo "Finished redeploying all components"

# Redeploy a component without rerunning the installer.
redeploy-%: push-to-gcr-% gen-yaml-%
	kubectl apply -f $(GEN_YAML_DIR)/$*.yaml

# Cleans all artifacts.
clean:
	rm -rf $(OUTPUT_DIR)

# Runs unit tests in a docker container.
test-unit: buildenv
	@docker run $(DOCKER_RUN_ARGS) ./scripts/test-unit.sh $(NOMOS_GO_PKG)

# Runs unit tests and linter.
test: test-unit lint

# Runs all tests.
test-all: test test-e2e

# Runs formatter.
goimports:
	goimports -w $(NOMOS_CODE_DIRS)

# Runs linter
lint: build
	@docker run $(DOCKER_RUN_ARGS) ./scripts/lint.sh $(NOMOS_GO_PKG)

# Generate K8S client-set.
gen-client-set:
	$(TOP_DIR)/scripts/generate-clientset.sh

# Installs Nomos kubectl plugin.
install-kubectl-plugin:
	cd $(TOP_DIR)/cmd/kubectl-nomos; ./install.sh

# Creates staging directory for generating docs.
docs-staging: $(OUTPUT_DIR)
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
