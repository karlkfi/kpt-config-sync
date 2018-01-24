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

# Directory containing templates yaml files.
TEMPLATES_DIR := $(TOP_DIR)/manifests/enrolled/templates

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

# Builds all go binaries statically.  These are good for deploying into very
# small containers, e.g. built off of "scratch" or "busybox" or some such.
.PHONY: build-all
build-all: .output
	CGO_ENABLED=0 GOBIN=$(BIN_DIR) \
	      go install -v -installsuffix="static" $(REPO)/cmd/...

# Builds all go packages.
.PHONY: build-alldeps
build-alldeps: .output
	CGO_ENABLED=0 GOBIN=$(BIN_DIR) \
	      go build -v $(REPO)/pkg/...

# Cleans all artifacts.
.PHONY: clean clean-all
clean: clean-all
clean-all:
	go clean -v $(REPO)
	rm -rf $(OUTPUT_DIR)

# Runs all tests.
.PHONY: test test-all
test: test-all
test-all:
	go test $(REPO)/...

# Runs end to end tests.
.PHONY: test-e2e
test-e2e: test
	e2e/e2e.sh

# Run all static analyzers.
.PHONY: lint
lint:
	gofmt -w $(STOLOS_CODE_DIRS)
	go tool vet $(STOLOS_CODE_DIRS)

# Installs Stolos kubectl plugin
.PHONY: install-kubectl-plugin
install-kubectl-plugin:
	cd $(TOP_DIR)/cmd/kubectl-stolos; ./install.sh

# Generates all yaml files.
# Note on m4: m4 does not expand anything after '#' in source file.
.PHONY: gen-all-yaml-files
gen-all-yaml-files: .output
	m4 -DIMAGE_NAME=gcr.io/$(GCP_PROJECT)/resourcequota-admission-controller:$(IMAGE_TAG) < \
	        $(TEMPLATES_DIR)/resourcequota-admission-controller.yaml > $(GEN_YAML_DIR)/resourcequota-admission-controller.yaml
	m4 -DIMAGE_NAME=gcr.io/$(GCP_PROJECT)/syncer:$(IMAGE_TAG) < $(TEMPLATES_DIR)/syncer.yaml > $(GEN_YAML_DIR)/syncer.yaml
	m4 -DIMAGE_NAME=gcr.io/$(GCP_PROJECT)/stolosresourcequota-controller:$(IMAGE_TAG) < \
	        $(TEMPLATES_DIR)/stolosresourcequota-controller.yaml > $(GEN_YAML_DIR)/stolosresourcequota-controller.yaml
	m4 -DIMAGE_NAME=gcr.io/$(GCP_PROJECT)/remote-cluster-policy-importer:$(IMAGE_TAG) < \
	        $(TEMPLATES_DIR)/remote-cluster-policy-importer.yaml > $(GEN_YAML_DIR)/remote-cluster-policy-importer.yaml

######################################################################
# Targets for building and pushing docker images.
######################################################################
.PHONY: push-resourcequota-admission-controller
push-resourcequota-admission-controller: build-all
	cp -r $(TOP_DIR)/build/resourcequota-admission-controller $(STAGING_DIR)
	cp $(BIN_DIR)/resourcequota-admission-controller $(STAGING_DIR)/resourcequota-admission-controller
	cd $(STAGING_DIR)/resourcequota-admission-controller; ./gencert.sh
	kubectl delete secret resourcequota-admission-controller-secret \
		--namespace=stolos-system || true
	kubectl create secret tls resourcequota-admission-controller-secret \
		--namespace=stolos-system \
	    --cert=$(STAGING_DIR)/resourcequota-admission-controller/server.crt \
		--key=$(STAGING_DIR)/resourcequota-admission-controller/server.key
	kubectl delete secret resourcequota-admission-controller-secret-ca \
		--namespace=stolos-system || true
	kubectl create secret tls resourcequota-admission-controller-secret-ca \
		--namespace=stolos-system \
	    --cert=$(STAGING_DIR)/resourcequota-admission-controller/ca.crt \
		--key=$(STAGING_DIR)/resourcequota-admission-controller/ca.key
	$(call build-and-push-image,resourcequota-admission-controller)

.PHONY: push-syncer
push-syncer: build-all
	cp -r $(TOP_DIR)/build/syncer $(STAGING_DIR)
	cp $(BIN_DIR)/syncer $(STAGING_DIR)/syncer
	$(call build-and-push-image,syncer)

.PHONY: push-stolosresourcequota-controller
push-stolosresourcequota-controller: build-all
	cp -r $(TOP_DIR)/build/stolosresourcequota-controller $(STAGING_DIR)
	cp $(BIN_DIR)/stolosresourcequota-controller $(STAGING_DIR)/stolosresourcequota-controller
	$(call build-and-push-image,stolosresourcequota-controller)

.PHONY: push-remote-cluster-policy-importer
push-remote-cluster-policy-importer: build-all
	cp -r $(TOP_DIR)/build/remote-cluster-policy-importer $(STAGING_DIR)
	cp $(BIN_DIR)/remote-cluster-policy-importer $(STAGING_DIR)/remote-cluster-policy-importer
	$(call build-and-push-image,remote-cluster-policy-importer)

######################################################################
# Targets for deploying components to K8S.
######################################################################
.PHONY: deploy-common-objects
deploy-common-objects:
	$(TOP_DIR)/scripts/deploy-common-objects.sh

.PHONY: deploy-resourcequota-admission-controller
deploy-resourcequota-admission-controller: push-resourcequota-admission-controller gen-all-yaml-files
	kubectl delete ValidatingWebhookConfiguration stolos-resource-quota --ignore-not-found
	kubectl replace -f $(GEN_YAML_DIR)/resourcequota-admission-controller.yaml --force
	$(TOP_DIR)/scripts/resourcequota-admission-controller/wait-for-endpoint.sh resourcequota-admission-controller

.PHONY: deploy-syncer
deploy-syncer: push-syncer gen-all-yaml-files
	kubectl replace -f $(GEN_YAML_DIR)/syncer.yaml --force

.PHONY: deploy-stolosresourcequota-controller
deploy-stolosresourcequota-controller: push-stolosresourcequota-controller gen-all-yaml-files
	kubectl replace -f $(GEN_YAML_DIR)/stolosresourcequota-controller.yaml --force

.PHONY: deploy-remote-cluster-policy-importer
deploy-remote-cluster-policy-importer: push-remote-cluster-policy-importer gen-all-yaml-files
	kubectl replace -f $(GEN_YAML_DIR)/remote-cluster-policy-importer.yaml --force

.PHONY: deploy-all
deploy-all: deploy-resourcequota-admission-controller deploy-syncer deploy-stolosresourcequota-controller
