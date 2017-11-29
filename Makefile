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

TOP_DIR := $(shell pwd)

REPO := github.com/google/stolos

# List of dirs containing go code owned by Stolos
STOLOS_CODE_DIRS := pkg cmd

# Directory containing all build artifacts.
OUTPUT_DIR := $(TOP_DIR)/.output

# Directory containing installed go binaries.
BIN_DIR := $(OUTPUT_DIR)/bin

# Directory used for staging Docker contexts.
STAGING_DIR := $(OUTPUT_DIR)/staging

# Directory containing gen-alld yaml files from manifest templates.
GEN_YAML_DIR := $(OUTPUT_DIR)/yaml

# Docker image tage.
IMAGE_TAG ?= $(USER)

GCP_PROJECT ?= stolos-dev

# Helper functions for building and pushing a docker image to gcr.io.
# Args:
#   $(1) Docker image name as well as directory name in staging (e.g. "resource-quota")
define build-and-push-image
	docker build -t gcr.io/$(GCP_PROJECT)/$(1):$(IMAGE_TAG) $(STAGING_DIR)/$(1)
	gcloud docker -- push gcr.io/$(GCP_PROJECT)/$(1):$(IMAGE_TAG)
endef

# Creates the local golang build output directory.
.output:
	mkdir -p $(OUTPUT_DIR)
	mkdir -p $(BIN_DIR)
	mkdir -p $(STAGING_DIR)
	mkdir -p $(GEN_YAML_DIR)

# Builds all go packages statically.  These are good for deploying into very
# small containers, e.g. built off of "scratch" or "busybox" or some such.
.PHONY: build-all
build-all: .output
	CGO_ENABLED=0 GOBIN=$(BIN_DIR) \
	      go install -v -installsuffix="static" $(REPO)/cmd/...

# Cleans all artifacts.
.PHONY: clean-all
clean-all:
	go clean -v $(REPO)
	rm -rf $(OUTPUT_DIR)

# Runs all tests.
.PHONY: test-all
test-all:
	go test -v $(REPO)/...

# Run all static analyzers.
.PHONY: lint
lint:
	gofmt -w $(STOLOS_CODE_DIRS)
	go tool vet $(STOLOS_CODE_DIRS)

# Generates all yaml files.
# Note on m4: m4 does not expand anything after '#' in source file.
.PHONY: gen-all-yaml-files
gen-all-yaml-files: .output
	m4 -DIMAGE_NAME=gcr.io/$(GCP_PROJECT)/resource-quota:$(IMAGE_TAG) < $(TOP_DIR)/manifests/templates/resource-quota.yaml > $(GEN_YAML_DIR)/resource-quota.yaml
	m4 -DIMAGE_NAME=gcr.io/$(GCP_PROJECT)/syncer:$(IMAGE_TAG) < $(TOP_DIR)/manifests/templates/syncer.yaml > $(GEN_YAML_DIR)/syncer.yaml
	m4 -DIMAGE_NAME=gcr.io/$(GCP_PROJECT)/authorizer:$(IMAGE_TAG) < $(TOP_DIR)/manifests/templates/authorizer.yaml > $(GEN_YAML_DIR)/authorizer.yaml

######################################################################
# Targets for building and pushing docker images.
######################################################################
.PHONY: push-resource-quota
push-resource-quota: build-all
	cp -r $(TOP_DIR)/deploy/admission-controller/resource-quota $(STAGING_DIR)
	cp $(BIN_DIR)/resource-quota $(STAGING_DIR)/resource-quota
	cd $(STAGING_DIR)/resource-quota; ./gencert.sh
	$(call build-and-push-image,resource-quota)

.PHONY: push-syncer
push-syncer: build-all
	cp -r $(TOP_DIR)/deploy/syncer $(STAGING_DIR)
	cp $(BIN_DIR)/syncer $(STAGING_DIR)/syncer
	$(call build-and-push-image,syncer)

.PHONY: push-authorizer
push-authorizer: build-all
	cp -r $(TOP_DIR)/deploy/authorizer $(STAGING_DIR)
	cp $(BIN_DIR)/authorizer $(STAGING_DIR)/authorizer
	cd $(STAGING_DIR)/authorizer; ./gencert.sh
	$(call build-and-push-image,authorizer)

######################################################################
# Targets for deploying components to K8S.
######################################################################
.PHONY: deploy-resource-quota
deploy-resource-quota: push-resource-quota gen-all-yaml-files
	kubectl replace -f $(GEN_YAML_DIR)/resource-quota.yaml --force

.PHONY: deploy-syncer
deploy-syncer: push-syncer gen-all-yaml-files
	kubectl replace -f $(GEN_YAML_DIR)/syncer.yaml --force

.PHONY: deploy-syncer
deploy-authorizer: push-authorizer gen-all-yaml-files
	kubectl replace -f $(GEN_YAML_DIR)/authorizer.yaml --force

# TODO: Add deploy to local target for authorizer here.

.PHONY: deploy-all
deploy-all: deploy-resource-quota deploy-syncer deploy-authorizer
