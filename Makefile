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

# Directory used for staging scripts.
SCRIPTS_STAGING_DIR := $(STAGING_DIR)/scripts

# Directory used for installer output.
INSTALLER_OUTPUT_DIR := $(STAGING_DIR)/installer_output

# Directory that gets mounted into /tmp for build and test containers.
TEMP_OUTPUT_DIR := $(OUTPUT_DIR)/tmp

# Directory containing gen-alld yaml files from manifest templates.
GEN_YAML_DIR := $(OUTPUT_DIR)/yaml

# Directory containing generated test yamls
TEST_GEN_YAML_DIR := $(OUTPUT_DIR)/test/yaml

# Directory containing templates yaml files.
TEMPLATES_DIR := $(TOP_DIR)/manifests/templates

# Directory containing test template yaml files.
TEST_TEMPLATES_DIR := $(TOP_DIR)/test/manifests/templates

# Use git tags to set version string.
VERSION := $(shell git describe --tags --always --dirty)

# Which architecture to build.
ARCH ?= amd64

# UID of current user
UID := $(shell id -u)

# GID of current user
GID := $(shell id -g)

# GCP project that owns container registry for local dev build.
GCP_PROJECT ?= stolos-dev

# Docker image used for build and test. This image does not support CGO.
# NOTE: nomos-public is fully accessible publicly, do not use for anything
# other than buildenv
BUILDENV_PROJECT ?= nomos-public
BUILDENV_IMAGE_VERSION ?= v0.1.0
BUILDENV_IMAGE ?= gcr.io/$(BUILDENV_PROJECT)/buildenv:$(BUILDENV_IMAGE_VERSION)

# GCP service account credentials used in e2e tests.
GCP_E2E_CRED := gs://stolos-dev/e2e/watcher_client_key.json

# When set to "release", enables these optimizations:
# - Compress binary sizes
BUILD_MODE ?= debug

# All Nomos K8S deployments.
ALL_K8S_DEPLOYMENTS := syncer \
	git-policy-importer \
	gcp-policy-importer \
	resourcequota-admission-controller \
	policy-admission-controller

# Nomos docker images containing all binaries.
NOMOS_IMAGE := nomos

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
IMAGE_TAG ?= $(VERSION)-$(USER)-$(DATE)

DOCKER_RUN_ARGS := \
	$(DOCKER_INTERACTIVE)                                              \
	-e ARCH=$(ARCH)                                                    \
	-e VERSION=$(VERSION)                                              \
	-e PKG=$(REPO)                                                     \
	-e BUILD_MODE=$(BUILD_MODE)                                        \
	-u $(UID):$(GID)                                                   \
	-v $(GO_DIR):/go                                                   \
	-v $$(pwd):/go/src/$(REPO)                                         \
	-v $(BIN_DIR)/$(ARCH):/go/bin                                      \
	-v $(GO_DIR)/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static    \
	-v $(TEMP_OUTPUT_DIR):/tmp                                         \
	-w /go/src/$(REPO)                                                 \
	--rm                                                               \
	$(BUILDENV_IMAGE)                                                  \

##### SETUP #####

.DEFAULT_GOAL := test

.PHONY: $(OUTPUT_DIR)
$(OUTPUT_DIR):
	@echo "+++ Creating the local build output directory: $(OUTPUT_DIR)"
	@mkdir -p \
		$(BIN_DIR)/$(ARCH) \
		$(DOCS_STAGING_DIR) \
		$(GEN_YAML_DIR) \
		$(GO_DIR)/pkg \
		$(GO_DIR)/src/$(REPO) \
		$(GO_DIR)/std/$(ARCH) \
		$(INSTALLER_OUTPUT_DIR) \
		$(SCRIPTS_STAGING_DIR) \
		$(STAGING_DIR) \
		$(TEMP_OUTPUT_DIR) \
		$(TEST_GEN_YAML_DIR)

##### TARGETS #####
PULL_BUILDENV := \
	docker image inspect $(BUILDENV_IMAGE) &> /dev/null \
	|| docker pull $(BUILDENV_IMAGE)
build-buildenv: build/buildenv/Dockerfile
	@echo "+++ Creating the buildenv docker container"
	@docker build build/buildenv -t $(BUILDENV_IMAGE)
	@gcloud $(GCLOUD_QUIET) auth configure-docker
	@docker push $(BUILDENV_IMAGE)

# The installer script is built by preprocessing a template script.
$(SCRIPTS_STAGING_DIR)/run-installer.sh: \
		$(OUTPUT_DIR) \
		$(TOP_DIR)/scripts/run-installer.sh.template \
		$(TOP_DIR)/scripts/lib/installer.sh
	@echo "+++ Building run-installer.sh"
	@docker run -it --rm -e PROGRAM=argbash \
	    -u $(UID):$(GID) \
  		-v "$(TOP_DIR):/work" matejak/argbash \
		/work/scripts/run-installer.sh.template \
		--output=$(SCRIPTS_STAGING_DIR:$(TOP_DIR)%=/work%)/run-installer.sh.1
	@cat \
		$(TOP_DIR)/scripts/lib/installer.sh \
		$(SCRIPTS_STAGING_DIR)/run-installer.sh.1 \
		> $@
	@sed -i -e "s/XXX_INSTALLER_DEFAULT_VERSION/${VERSION}/g" $@
	@chmod +x $@

$(SCRIPTS_STAGING_DIR)/nomosvet.sh: \
		$(OUTPUT_DIR) \
		$(TOP_DIR)/scripts/nomosvet.sh
	@echo "+++ Building nomosvet.sh"
	@cp $(TOP_DIR)/scripts/nomosvet.sh $@
	@sed -i -e "s/XXX_NOMOSVET_DEFAULT_VERSION/${VERSION}/g" $@

.PHONY: build
build: $(OUTPUT_DIR)
	@echo "+++ Compiling Nomos binaries hermetically using a docker container"
	@$(PULL_BUILDENV)
	@docker run $(DOCKER_RUN_ARGS) ./scripts/build.sh

# Creates a docker image for the specified nomos component.
image-nomos: build
	@echo "+++ Building the Nomos image"
	@cp -r $(TOP_DIR)/build/$(NOMOS_IMAGE) $(STAGING_DIR)
	@cp $(BIN_DIR)/$(ARCH)/* $(STAGING_DIR)/$(NOMOS_IMAGE)
	@docker build $(DOCKER_BUILD_QUIET) \
			-t gcr.io/$(GCP_PROJECT)/$(NOMOS_IMAGE):$(IMAGE_TAG) $(STAGING_DIR)/$(NOMOS_IMAGE)

GIT_SERVER_DOCKER := $(OUTPUT_DIR)/git-server-docker
GIT_SERVER_RELEASE := v1.0.0
# Creates docker image for the test git-server from github source
build-git-server:
	@echo "+++ Building image for test git server"
	@mkdir -p $(OUTPUT_DIR)
	@rm -rf $(GIT_SERVER_DOCKER)
	@git clone https://github.com/jkarlosb/git-server-docker.git $(GIT_SERVER_DOCKER)
	@cd $(GIT_SERVER_DOCKER) && git checkout $(GIT_SERVER_RELEASE)
	@docker build $(DOCKER_BUILD_QUIET) \
			$(GIT_SERVER_DOCKER) \
			-t gcr.io/$(GCP_PROJECT)/git-server:$(GIT_SERVER_RELEASE)
	@gcloud $(GCLOUD_QUIET) auth configure-docker
	@docker push gcr.io/$(GCP_PROJECT)/git-server:$(GIT_SERVER_RELEASE)

# Pushes the specified component's docker image to gcr.io.
push-to-gcr-%: image-%
	@echo "+++ Pushing $* to gcr.io"
	@gcloud $(GCLOUD_QUIET) auth configure-docker
	@docker push gcr.io/$(GCP_PROJECT)/$*:$(IMAGE_TAG)

# Generates the podspec yaml for each component.
gen-yaml-all: $(addprefix gen-yaml-, $(ALL_K8S_DEPLOYMENTS))
	@echo "+++ Finished generating all yaml"

# Generates the podspec yaml for the component specified.
gen-yaml-%:
	@echo "+++ Generating yaml $*"
	@m4 -DIMAGE_NAME=gcr.io/$(GCP_PROJECT)/$(NOMOS_IMAGE):$(IMAGE_TAG) < \
			$(TEMPLATES_DIR)/$*.yaml > $(GEN_YAML_DIR)/$*.yaml

# Generates yaml for the test git server
$(TEST_GEN_YAML_DIR)/git-server.yaml: $(TEST_TEMPLATES_DIR)/git-server.yaml Makefile $(OUTPUT_DIR)
	@echo "+++ Generating yaml git-server"
	@m4 -DIMAGE_NAME=gcr.io/$(GCP_PROJECT)/git-server:$(GIT_SERVER_RELEASE) < \
			 $< > $@

# Creates staging directory for building installer docker image.
# TODO(filmil): Depending on push-to-gcr-all here seems unnecessary.
installer-staging: push-to-gcr-nomos gen-yaml-all $(OUTPUT_DIR)
	@echo "+++ Creating staging directory for building installer docker image"
	@cp -r $(TOP_DIR)/build/installer $(STAGING_DIR)
	@cp $(BIN_DIR)/$(ARCH)/installer $(STAGING_DIR)/installer
	@mkdir -p $(STAGING_DIR)/installer/yaml
	@cp $(OUTPUT_DIR)/yaml/*  $(STAGING_DIR)/installer/yaml
	@mkdir -p $(STAGING_DIR)/installer/manifests
	@cp $(TOP_DIR)/manifests/*.yaml $(STAGING_DIR)/installer/manifests
	@mkdir -p $(STAGING_DIR)/installer/examples
	@cp -r $(TOP_DIR)/build/installer/examples/* $(STAGING_DIR)/installer/examples
	@cp $(TOP_DIR)/build/installer/entrypoint.sh $(STAGING_DIR)/installer
	@mkdir -p $(STAGING_DIR)/installer/scripts
	@cp $(TOP_DIR)/scripts/deploy-resourcequota-admission-controller.sh \
		$(STAGING_DIR)/installer/scripts
	@cp $(TOP_DIR)/scripts/generate-admission-controller-certs.sh \
		$(STAGING_DIR)/installer/scripts
	@cp $(TOP_DIR)/scripts/generate-resourcequota-admission-controller-certs.sh \
		$(STAGING_DIR)/installer/scripts
	@cp $(TOP_DIR)/scripts/deploy-policy-admission-controller.sh \
		$(STAGING_DIR)/installer/scripts
	@cp $(TOP_DIR)/scripts/generate-policy-admission-controller-certs.sh \
		$(STAGING_DIR)/installer/scripts

# Builds the installer docker image using the nomos release in $(OUTPUT_DIR)
image-installer: installer-staging
	@echo "+++ Building the installer docker image using the Nomos release in $(OUTPUT_DIR)"
	@docker build $(DOCKER_BUILD_QUIET) \
			-t gcr.io/$(GCP_PROJECT)/installer:test-e2e-latest \
			-t gcr.io/$(GCP_PROJECT)/installer:$(IMAGE_TAG) \
     		--build-arg "INSTALLER_VERSION=$(IMAGE_TAG)" \
		$(STAGING_DIR)/installer

# Runs the installer via docker in interactive mode.
install-interactive: deploy-interactive
deploy-interactive: $(SCRIPTS_STAGING_DIR)/run-installer.sh image-installer
	$(SCRIPTS_STAGING_DIR)/run-installer.sh \
		--interactive \
		--container=gcr.io/$(GCP_PROJECT)/installer \
		--output_dir=$(INSTALLER_OUTPUT_DIR) \
		--version=$(IMAGE_TAG)

check-nomos-installer-config:
	@echo "+++ Checking installer configuration"
	@if [ -z "$(NOMOS_INSTALLER_CONFIG)" ]; then \
		echo '### Must set env variable NOMOS_INSTALLER_CONFIG to use make deploy'; \
		exit 1; \
	fi;

# Runs the installer via docker in batch mode using the installer config
# file specied in the environment variable NOMOS_INSTALLER_CONFIG.
install: deploy
deploy: check-nomos-installer-config \
		$(SCRIPTS_STAGING_DIR)/run-installer.sh image-installer
	@echo "+++ Running installer with output directory: $(INSTALLER_OUTPUT_DIR)"
	@$(SCRIPTS_STAGING_DIR)/run-installer.sh \
		--config=$(NOMOS_INSTALLER_CONFIG) \
		--container=gcr.io/$(GCP_PROJECT)/installer \
		--output_dir=$(INSTALLER_OUTPUT_DIR) \
		--use_current_context=true \
		--version=$(IMAGE_TAG)

# Runs uninstallation via docker in batch mode using the installer config.
uninstall: check-nomos-installer-config \
		$(SCRIPTS_STAGING_DIR)/run-installer.sh image-installer
	@echo "+++ Running installer with output directory: $(INSTALLER_OUTPUT_DIR)"
	@$(SCRIPTS_STAGING_DIR)/run-installer.sh \
		--config=$(NOMOS_INSTALLER_CONFIG) \
		--container=gcr.io/$(GCP_PROJECT)/installer \
		--output_dir=$(INSTALLER_OUTPUT_DIR) \
		--use_current_context=true \
		--uninstall=deletedeletedelete \
		--version=$(IMAGE_TAG)

# Note that it is basically a copy of the staging directory for the installer
# plus a few extras for e2e.
e2e-staging: $(TEST_GEN_YAML_DIR)/git-server.yaml $(SCRIPTS_STAGING_DIR)/run-installer.sh $(OUTPUT_DIR)
	@echo "+++ Creating staging directory for building e2e docker image"
	@mkdir -p $(STAGING_DIR)/e2e-tests $(OUTPUT_DIR)/e2e
	@cp -r $(TOP_DIR)/build/e2e-tests/* $(STAGING_DIR)/e2e-tests
	@cp -r $(TOP_DIR)/third_party/bats-core $(OUTPUT_DIR)/e2e/bats
	@cp -r $(TOP_DIR)/examples/acme/sot $(OUTPUT_DIR)/e2e
	@cp -n $(HOME)/.ssh/id_rsa.nomos $(OUTPUT_DIR)/e2e/id_rsa.nomos
	@cp $(HOME)/.ssh/id_rsa.nomos.pub $(OUTPUT_DIR)/e2e/id_rsa.nomos.pub
	@cp $(TOP_DIR)/scripts/init-git-server.sh $(OUTPUT_DIR)/e2e/
	@cp $(TEST_GEN_YAML_DIR)/git-server.yaml $(OUTPUT_DIR)/e2e/
	@cp $(SCRIPTS_STAGING_DIR)/run-installer.sh $(OUTPUT_DIR)/e2e/

# Builds the e2e docker image and depencencies.
# Note that the GCP project is hardcoded since we currently don't want
# or need a nomos-release version of the e2e-tests image.
e2e-image: e2e-staging
	@echo "+++ Building the e2e docker image"
	@docker build $(DOCKER_BUILD_QUIET) \
		-t gcr.io/stolos-dev/e2e-tests:test-e2e-latest \
		--build-arg "DOCKER_GID=$(shell stat -c '%g' /var/run/docker.sock)" \
		--build-arg "UID=$(UID)" \
		--build-arg "GID=$(GID)" \
		--build-arg "UNAME=$(USER)" \
		$(STAGING_DIR)/e2e-tests

e2e-image-all: e2e-image image-installer

# Runs e2e tests for a particular importer.
GCLOUD_PATH = $(dir $(shell which gcloud))
KUBECTL_PATH = $(dir $(shell which kubectl))

# Hack around b/111407514 for goobuntu installed with standard
# gcloud/kubectl packages.
ifeq ($(KUBECTL_PATH),/usr/bin/)
	KUBECTL_E2E_MOUNT := -v "/usr/bin/kubectl:/usr/bin/kubectl"
endif
ifeq ($(GCLOUD_PATH),/usr/bin/)
	GCLOUD_E2E_MOUNT := \
	  -v "/usr/lib/google-cloud-sdk:/usr/lib/google-cloud-sdk"
	GCLOUD_PATH := /usr/lib/google-cloud-sdk/bin
endif

test-e2e-run-%:
	@echo "+++ Running $* e2e tests"
	@mkdir -p ${INSTALLER_OUTPUT_DIR}/{kubeconfig,certs,gen_configs,logs}
	@rm -rf $(OUTPUT_DIR)/e2e/testcases
	@cp -r $(TOP_DIR)/e2e $(OUTPUT_DIR)
	docker run -it \
	    --group-add docker \
	    -u $(UID):$(GID) \
	    -v /var/run/docker.sock:/var/run/docker.sock \
	    -v "$(HOME)":$(HOME) \
	    -v "$(OUTPUT_DIR)/e2e":/opt/testing/e2e \
		$(KUBECTL_E2E_MOUNT) \
		$(GCLOUD_E2E_MOUNT) \
		$(E2E_AUX_VOLUMES) \
	    -e "CONTAINER=gcr.io/$(GCP_PROJECT)/installer" \
	    -e "VERSION=$(IMAGE_TAG)" \
	    -e "HOME=$(HOME)" \
	    -e "GCLOUD_PATH=$(GCLOUD_PATH)" \
	    -e "KUBECTL_PATH=$(KUBECTL_PATH)" \
	    -e "NOMOS_REPO=$(shell pwd)" \
	    "gcr.io/stolos-dev/e2e-tests:test-e2e-latest" \
	    ${E2E_FLAGS} \
		--importer $* \
		--gcp-cred "$(GCP_E2E_CRED)" \
	    && echo "+++ $* e2e tests completed" \
	    || (echo "### e2e tests failed. Logs are available in ${INSTALLER_OUTPUT_DIR}/logs"; exit 1)

E2E_PARAMS := \
	IMAGE_TAG=$(IMAGE_TAG) \
	GCP_PROJECT=$(GCP_PROJECT) \
	GCP_E2E_CRED=$(GCP_E2E_CRED)

# Clean, build, and run e2e tests for all importers.
# Clean cluster after running.
test-e2e-all: clean e2e-image-all
	$(MAKE) test-e2e-run-git \
		$(E2E_PARAMS) \
		E2E_FLAGS="--clean --test --setup $(E2E_FLAGS)"
	$(MAKE) test-e2e-run-gcp \
		$(E2E_PARAMS) \
		E2E_FLAGS="--clean --test --setup $(E2E_FLAGS)"

# Clean, build, and run e2e tests for a particular importer.
# Clean cluster after running.
test-e2e-%: clean e2e-image-all
	$(MAKE) test-e2e-run-$* \
		$(E2E_PARAMS) \
		E2E_FLAGS="--clean --test --setup $(E2E_FLAGS)"

# Clean, build, and run e2e tests for a parcular importer.
# Do not clean cluster after running so that the state can be investigated.
test-e2e-noclean-%: e2e-image-all
	$(MAKE) test-e2e-run-$* \
		$(E2E_PARAMS) \
		E2E_FLAGS="--test --setup $(E2E_FLAGS)"

# Run e2e tests but do not build. Assumes that the necesary images have already been built and pushed,
# so `make test-e2e` should be run before this, and each time the system requires a rebuild as this
# target has no intelligence about dependencies.
test-e2e-nosetup-%:
	$(MAKE) test-e2e-run-$* \
		$(E2E_PARAMS) \
		E2E_FLAGS="--test --clean $(E2E_FLAGS)"

# Dev mode allows for specifying the args manually via E2E_FLAGS, see e2e/setup.sh for
# allowed flags
test-e2e-dev-git:
	$(MAKE) test-e2e-run-git \
		GCP_PROJECT=$(GCP_PROJECT) \
		IMAGE_TAG=test-e2e-latest

# Redeploy a component without rerunning the installer.
redeploy-%: push-to-gcr-nomos gen-yaml-%
	@echo "+++ Redeploying without rerunning the installer: $*"
	@kubectl apply -f $(GEN_YAML_DIR)/$*.yaml

# Cleans all artifacts.
clean:
	@echo "+++ Cleaning $(OUTPUT_DIR)"
	@rm -rf $(OUTPUT_DIR)

test-unit: $(OUTPUT_DIR)
	@echo "+++ Running unit tests in a docker container"
	@$(PULL_BUILDENV)
	@docker run $(DOCKER_RUN_ARGS) ./scripts/test-unit.sh $(NOMOS_GO_PKG)

# Runs unit tests and linter.
test: test-unit lint lint-bash

# Runs all tests.
test-all: test test-e2e-all

goimports:
	@echo "+++ Running goimports"
	@goimports -w $(NOMOS_CODE_DIRS)

lint: build
	@docker run $(DOCKER_RUN_ARGS) ./scripts/lint.sh $(NOMOS_GO_PKG)

lint-bash:
	@./scripts/lint-bash.sh

.PHONY: clientgen
clientgen:
	@echo "+++ Generating clientgen directory"
	rm -rf $(TOP_DIR)/clientgen
	$(TOP_DIR)/scripts/generate-clientset.sh
	$(TOP_DIR)/scripts/generate-watcher.sh

# Installs Nomos kubectl plugin.
install-kubectl-plugin:
	@echo "+++ Installing Nomos kubectl plugin"
	@cd $(TOP_DIR)/cmd/kubectl-nomos; ./install.sh

# Creates staging directory for generating docs.
docs-staging: $(OUTPUT_DIR)
	@echo "+++ Creating directory for generating docs: $(OUTPUT_DIR)"
	@rm -rf $(DOCS_STAGING_DIR)/*.html $(DOCS_STAGING_DIR)/img
	@cp -r $(TOP_DIR)/docs $(STAGING_DIR)
	@cp $(TOP_DIR)/README.md $(DOCS_STAGING_DIR)

docs-generate: $(OUTPUT_DIR) docs-staging
	@echo "+++ Converting Markdown docs into HTML"
	@$(PULL_BUILDENV)
	@docker run $(DOCKER_INTERACTIVE) \
		-u $(UID):$(GID)                                                   \
		-v $$(pwd):/go/src/$(REPO)                                         \
		-w /go/src/$(REPO)                                                 \
		-e HOME=/go/src/$(REPO)/.output/config                             \
		-e REPO=$(REPO)                                                    \
		-e DOCS_STAGING_DIR=/go/src/$(REPO)/.output/staging/docs           \
		-e TOP_DIR=/go/src/$(REPO)                                         \
		--rm                                                               \
		$(BUILDENV_IMAGE)                                                  \
		/bin/bash -c " \
		    ./scripts/docs-generate.sh \
	    "
	@echo "Open in your browser: file://$(DOCS_STAGING_DIR)/README.html"

docs-package: docs-generate
	@echo "+++ Packaging HTML docs into a zip package"
	@cd $(STAGING_DIR); rm -f nomos-docs.zip; zip -r nomos-docs.zip docs

docs-format:
	@/google/data/ro/teams/g3doc/mdformat --in_place --compatibility $(shell find docs README.md -name '*.md')

