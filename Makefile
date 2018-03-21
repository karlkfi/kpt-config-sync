#!/usr/bin/make
#
# Copyright 2018 Stolos Authors
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

TOP_DIR := $(shell pwd)

REPO := github.com/google/nomos

# List of dirs containing go code owned by Stolos
STOLOS_CODE_DIRS := pkg cmd

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

# All Stolos components
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

# Creates the local golang build output directory.
.output:
	mkdir -p \
		$(GO_DIR)/src/$(REPO) \
		$(GO_DIR)/pkg \
		$(GO_DIR)/std/$(ARCH) \
		$(BIN_DIR)/$(ARCH) \
		$(STAGING_DIR) \
		$(GEN_YAML_DIR)

# Uses a custom build environment.  See toolkit/buildenv/Dockerfile for the
# details on the build environment.
.PHONY: buildenv
buildenv:
	@docker build toolkit/buildenv --tag=$(BUILD_IMAGE)

# Build packages hermetically in a docker container.
.PHONY: build all-build
build: all-build
all-build: buildenv .output
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

# Cleans all artifacts.
clean: all-clean
all-clean:
	rm -rf $(OUTPUT_DIR)

# Runs unit tests in a docker container.
test: all-test
all-test: buildenv .output
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
			./scripts/build/test.sh $(STOLOS_CODE_DIRS)                    \
		"

# Runs end-to-end tests.
test-e2e:
	e2e/e2e.sh

test-e2e-no-cleanup:
	e2e/e2e.sh -skip_cleanup

# Runs all static analyzers and autocorrects.
lint:
	goimports -w $(STOLOS_CODE_DIRS)

# Generate K8S client-set.
gen-client-set:
	$(TOP_DIR)/scripts/generate-clientset.sh

# Installs Stolos kubectl plugin.
install-kubectl-plugin:
	cd $(TOP_DIR)/cmd/kubectl-stolos; ./install.sh

# Generates yaml from template.
gen-yaml-%:
	m4 -DIMAGE_NAME=gcr.io/$(GCP_PROJECT)/$*:$(IMAGE_TAG) < \
			$(TEMPLATES_DIR)/$*.yaml > $(GEN_YAML_DIR)/$*.yaml

# Pushes the stolos installer (effectively releasing a new version!)
push-installer: GCP_PROJECT=nomos-release
push-installer: all-push
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

# Builds and pushes a docker image.
push-%: build
	@echo
	@echo "******* Pushing $* *******"
	@echo
	cp -r $(TOP_DIR)/build/$* $(STAGING_DIR)
	cp $(BIN_DIR)/$(ARCH)/$* $(STAGING_DIR)/$*
	docker build -t gcr.io/$(GCP_PROJECT)/$*:$(IMAGE_TAG) $(STAGING_DIR)/$*
	gcloud docker -- push gcr.io/$(GCP_PROJECT)/$*:$(IMAGE_TAG)

all-push: $(addprefix push-, $(ALL_COMPONENTS))

# Generates the yaml deployment from template and applies it to cluster.
deploy-%:	push-% gen-yaml-%
	kubectl apply -f $(GEN_YAML_DIR)/$*.yaml

deploy-common-objects:
	$(TOP_DIR)/scripts/deploy-common-objects.sh

gen-certs-resourcequota-admission-controller:
	OUTPUT_DIR=$(STAGING_DIR)/resourcequota-admission-controller \
	    $(TOP_DIR)/scripts/generate-resourcequota-admission-controller-certs.sh

deploy-resourcequota-admission-controller: \
	push-resourcequota-admission-controller \
	gen-yaml-resourcequota-admission-controller \
	gen-certs-resourcequota-admission-controller
	CERTS_INPUT_DIR=$(STAGING_DIR)/resourcequota-admission-controller \
	YAML_DIR=$(GEN_YAML_DIR) \
	    $(TOP_DIR)/scripts/deploy-resourcequota-admission-controller.sh

all-deploy: $(addprefix deploy-, $(ALL_COMPONENTS))

