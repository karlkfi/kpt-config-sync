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

.DEFAULT_GOAL := test

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
		$(DOCS_STAGING_DIR) \
		$(GEN_YAML_DIR) \
		$(TEMP_OUTPUT_DIR) \
		$(TEST_GEN_YAML_DIR)

##### TARGETS #####
# Pulls the cached builenv docker image from gcrio.
pull-buildenv:
	@docker image inspect $(BUILDENV_IMAGE) &> /dev/null \
	|| docker pull $(BUILDENV_IMAGE)

build-buildenv: build/buildenv/Dockerfile
	@echo "+++ Creating the buildenv docker container"
	@docker build build/buildenv -t $(BUILDENV_IMAGE)
	@gcloud $(GCLOUD_QUIET) auth configure-docker
	@docker push $(BUILDENV_IMAGE)

# Compiles binaries for a specific platform (e.g. "linux-amd64").
build-platform-%:
	@echo "+++ Compiling Nomos binaries for $* platform"
	@docker run -e PLATFORM=$* $(DOCKER_RUN_ARGS) ./scripts/build.sh

.PHONY: build
build: $(OUTPUT_DIR) pull-buildenv $(addprefix build-platform-, $(PLATFORMS))
	@echo "+++ Finished building"

# Creates a docker image for the specified nomos component.
image-nomos: build
	@echo "+++ Building the Nomos image"
	@cp -r $(TOP_DIR)/build/$(NOMOS_IMAGE) $(STAGING_DIR)
	@cp $(BIN_DIR)/linux_amd64/* $(STAGING_DIR)/$(NOMOS_IMAGE)
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
	@echo "+++ Using account:"
	gcloud config get-value account
	@gcloud $(GCLOUD_QUIET) auth configure-docker
	@docker push gcr.io/$(GCP_PROJECT)/$*:$(IMAGE_TAG)

# Generates the podspec yaml for each component.
gen-yaml-all: $(addprefix gen-yaml-, $(ALL_K8S_DEPLOYMENTS))
	@echo "+++ Finished generating all yaml"

# Generates the podspec yaml for the component specified.
gen-yaml-%:
	@echo "+++ Generating yaml $*"
	@sed -e 's|IMAGE_NAME|gcr.io/$(GCP_PROJECT)/$(NOMOS_IMAGE):$(IMAGE_TAG)|' < \
			$(TEMPLATES_DIR)/$*.yaml > $(GEN_YAML_DIR)/$*.yaml

$(TEST_GEN_YAML_DIR)/git-server.yaml: $(TEST_TEMPLATES_DIR)/git-server.yaml Makefile $(OUTPUT_DIR)
	@echo "+++ Generating yaml for git-server"
	@sed -e 's|IMAGE_NAME|gcr.io/$(GCP_PROJECT)/git-server:$(GIT_SERVER_RELEASE)|' < \
			 $< > $@

installer-staging: push-to-gcr-nomos gen-yaml-all $(OUTPUT_DIR)
	@echo "+++ Creating installer staging directory"
	@( \
			cd $(GO_DIR); \
			rsync --relative --quiet \
			      `find -name '*' -type f | grep installer` \
			      $(STAGING_DIR)/installer/; \
	)
	@cp -r $(TOP_DIR)/build/installer $(STAGING_DIR)
	@mkdir -p $(STAGING_DIR)/installer/yaml
	@cp $(OUTPUT_DIR)/yaml/*  $(STAGING_DIR)/installer/yaml
	@mkdir -p $(STAGING_DIR)/installer/manifests
	@cp $(TOP_DIR)/manifests/*.yaml $(STAGING_DIR)/installer/manifests
	@cp $(TOP_DIR)/build/installer/install.sh $(STAGING_DIR)/installer
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

check-nomos-installer-config:
	@echo "+++ Checking installer configuration"
	@if [ -z "$(NOMOS_INSTALLER_CONFIG)" ]; then \
		echo '### Must set env variable NOMOS_INSTALLER_CONFIG to use make deploy'; \
		exit 1; \
	fi;

# Note that it is basically a copy of the staging directory for the installer
# plus a few extras for e2e.
e2e-staging: \
		$(OUTPUT_DIR) \
		$(TEST_GEN_YAML_DIR)/git-server.yaml \
		installer-staging \
		$(wildcard $(TOP_DIR)/e2e/*)
	@echo "+++ Creating staging directory for building e2e docker image"
	@mkdir -p $(STAGING_DIR)/e2e-tests $(OUTPUT_DIR)/e2e
	@cp -r $(TOP_DIR)/build/e2e-tests/* $(STAGING_DIR)/e2e-tests
	@cp -r $(TOP_DIR)/third_party/bats-core $(OUTPUT_DIR)/e2e/bats
	@cp -r $(TOP_DIR)/examples/acme/sot $(OUTPUT_DIR)/e2e
	@if [ -f $(HOME)/.ssh/id_rsa.nomos ]; then \
	  cp -n $(HOME)/.ssh/id_rsa.nomos $(OUTPUT_DIR)/e2e/id_rsa.nomos; \
	  cp $(HOME)/.ssh/id_rsa.nomos.pub $(OUTPUT_DIR)/e2e/id_rsa.nomos.pub; \
	fi
	@cp $(TOP_DIR)/scripts/init-git-server.sh $(OUTPUT_DIR)/e2e/
	@cp $(TEST_GEN_YAML_DIR)/git-server.yaml $(OUTPUT_DIR)/e2e/
	@cp -r $(STAGING_DIR)/installer $(OUTPUT_DIR)/e2e/
	@cp -r $(TOP_DIR)/e2e $(OUTPUT_DIR)

# Builds the e2e docker image and depencencies.
# Note that the GCP project is hardcoded since we currently don't want
# or need a nomos-release version of the e2e-tests image.
image-e2e-tests: e2e-staging
	@echo "+++ Building the e2e docker image"
	@build/e2e-tests/build.sh \
		-t gcr.io/stolos-dev/e2e-tests:$(IMAGE_TAG) \
		-t gcr.io/stolos-dev/e2e-tests:test-e2e-latest \
		$(DOCKER_BUILD_QUIET)

image-e2e-prober: image-e2e-tests build/e2e-tests/Dockerfile.prober
	@echo "+++ Building the e2e prober image"
	@docker build \
	  --file=build/e2e-tests/Dockerfile.prober \
	  --build-arg "DOCKER_GID=$(shell stat -c '%g' /var/run/docker.sock)" \
	  --build-arg "UID=$(UID)" \
	  --build-arg "GID=$(GID)" \
	  --build-arg "UNAME=${USER}" \
	  -t gcr.io/stolos-dev/e2e-prober:$(IMAGE_TAG) \
	  -t gcr.io/stolos-dev/e2e-prober:test-e2e-latest \
	  $(DOCKER_BUILD_QUIET) \
	  .

test-e2e-prober: image-e2e-prober scripts/run-prober.sh
	@echo "+++ Running prober e2e test in a local docker container."
	@./scripts/run-prober.sh

e2e-image-all: image-e2e-tests installer-staging

test-e2e-run-%:
	@echo "+++ Running e2e tests: $*"
	@mkdir -p ${INSTALLER_OUTPUT_DIR}/{kubeconfig,certs,gen_configs,logs}
	@e2e/e2e.sh \
		--TEMP_OUTPUT_DIR "$(TEMP_OUTPUT_DIR)" \
		--OUTPUT_DIR "$(OUTPUT_DIR)" \
		$(TEST_E2E_RUN_FLAGS) \
		-- \
		--importer "$*" \
		--gcp-watcher-cred "$(GCP_E2E_WATCHER_CRED)" \
		--gcp-runner-cred "$(GCP_E2E_RUNNER_CRED)" \
		$(E2E_FLAGS) \
	    && echo "+++ $* e2e tests completed" \
	    || (echo "### e2e tests failed. Temp dir (with test output logs etc) are available in ${TEMP_OUTPUT_DIR}"; exit 1)

E2E_PARAMS := \
	IMAGE_TAG=$(IMAGE_TAG) \
	GCP_PROJECT=$(GCP_PROJECT) \
	GCP_E2E_WATCHER_CRED=$(GCP_E2E_WATCHER_CRED) \
        GCP_E2E_RUNNER_CRED=$(GCP_E2E_RUNNER_CRED)

# Clean, build, and run e2e tests for all importers.
# Clean cluster after running.
test-e2e-all: e2e-image-all
	$(MAKE) test-e2e-run-git \
		$(E2E_PARAMS) \
		E2E_FLAGS="--preclean --setup --test --clean $(E2E_FLAGS)" \
		TEST_E2E_RUN_FLAGS="$(TEST_E2E_RUN_FLAGS)"
	$(MAKE) test-e2e-run-gcp \
		$(E2E_PARAMS) \
		E2E_FLAGS="--preclean --setup --test --clean $(E2E_FLAGS)" \
		TEST_E2E_RUN_FLAGS="$(TEST_E2E_RUN_FLAGS)"

# Target intended for running the e2e tests with installation as a self-contained
# setup using a custom identity file.  The identity file is mounted in by the
# test runner.
ci-test-e2e:
	$(MAKE) $(E2E_PARAMS) \
      E2E_FLAGS="\
        --tap \
		--create-ssh-key \
        --gcp-prober-cred /tmp/config/prober_runner_client_key.json \
		$(E2E_FLAGS) \
     " \
     TEST_E2E_RUN_FLAGS="\
	    --hermetic \
	 " \
     test-e2e-all

# Same target as above, except set up so that it can be ran from a local machine.
# We must download the gcs prober credentials, as they will not be mounted in.
ci-test-e2e-local:
	$(MAKE) $(E2E_PARAMS) \
      E2E_FLAGS="\
        --tap \
		--create-ssh-key \
        --gcp-prober-cred /tmp/config/prober_runner_client_key.json \
		$(E2E_FLAGS) \
     " \
     TEST_E2E_RUN_FLAGS="\
	    --hermetic \
		--gcs-prober-cred gs://stolos-dev/e2e/nomos-e2e.joonix.net/prober_runner_client_key.json\
	 " \
     ci-test-e2e
# Clean, build, and run e2e tests for a particular importer.
# Clean cluster after running.
test-e2e-%: e2e-image-all
	$(MAKE) test-e2e-run-$* \
		$(E2E_PARAMS) \
		E2E_FLAGS="--preclean --setup --test --clean $(E2E_FLAGS)"

test-e2e-dev-gcp:
test-e2e-dev-git:
# Dev mode allows for specifying the args manually via E2E_FLAGS, see e2e/setup.sh for
# allowed flags
test-e2e-dev-%:
	$(MAKE) test-e2e-run-$* \
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

test-unit: $(OUTPUT_DIR) pull-buildenv
	@echo "+++ Running unit tests in a docker container"
	@docker run $(DOCKER_RUN_ARGS) ./scripts/test-unit.sh $(NOMOS_GO_PKG)

# Runs unit tests and linter.
test: test-unit lint lint-bash

# Runs all tests.
test-all: test test-e2e-all

goimports:
	@echo "+++ Running goimports"
	@goimports -w $(NOMOS_CODE_DIRS)

lint: build lint-bash
	@docker run $(DOCKER_RUN_ARGS) ./scripts/lint.sh $(NOMOS_GO_PKG)

lint-bash:
	@./scripts/lint-bash.sh

.PHONY: clientgen
clientgen:
	@echo "+++ Generating clientgen directory"
	$(TOP_DIR)/scripts/generate-clientset.sh

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

