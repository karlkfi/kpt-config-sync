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

# Use git tags to set version string.
VERSION := $(shell git describe --tags --always --dirty)

# Whether this is an official release or dev workflow.
RELEASE ?= 0

# GCP project that owns container registry.
GCP_PROJECT ?= stolos-dev

# All Stolos components
ALL_COMPONENTS := syncer \
	git-policy-importer \
	remote-cluster-policy-importer \
	resourcequota-admission-controller

# Docker image tag.
ifeq ($(RELEASE), 1)
	IMAGE_TAG := $(VERSION)
else
	IMAGE_TAG := $(VERSION)-$(shell date +'%s')
endif

# Creates the local golang build output directory.
.output:
	mkdir -p \
		$(OUTPUT_DIR) \
		$(BIN_DIR) \
		$(STAGING_DIR) \
		$(GEN_YAML_DIR)

# Builds all go binaries statically.  These are good for deploying into very
# small containers, e.g. built off of "scratch" or "busybox" or some such.
all-build: .output
	CGO_ENABLED=0 GOBIN=$(BIN_DIR) \
	      go install -v -installsuffix="static" $(REPO)/cmd/...

# Builds all go packages.
all-build-deps: .output
	CGO_ENABLED=0 GOBIN=$(BIN_DIR) \
	      go build -v $(REPO)/pkg/...

# Cleans all artifacts.
clean: all-clean
all-clean:
	go clean -v $(REPO)
	rm -rf $(OUTPUT_DIR)

# Runs all tests.
# TODO(b/73018918): Add test-e2e when it's not flaky.
all-test: test-unit

# Runs unit tests.
test-unit:
	go test $(REPO)/...

# Runs end-to-end tests.
test-e2e:
	e2e/e2e.sh

# Runs all static analyzers and autocorrects.
lint:
	goimports -w $(STOLOS_CODE_DIRS)

# Installs Stolos kubectl plugin.
install-kubectl-plugin:
	cd $(TOP_DIR)/cmd/kubectl-stolos; ./install.sh

# Generates yaml from template.
gen-yaml-%:
	m4 -DIMAGE_NAME=gcr.io/$(GCP_PROJECT)/$*:$(IMAGE_TAG) < \
	        $(TEMPLATES_DIR)/$*.yaml > $(GEN_YAML_DIR)/$*.yaml

# Builds and pushes a docker image.
push-%: all-build
	@echo
	@echo "******* Pushing $* *******"
	@echo
	cp -r $(TOP_DIR)/build/$* $(STAGING_DIR)
	cp $(BIN_DIR)/$* $(STAGING_DIR)/$*
	docker build -t gcr.io/$(GCP_PROJECT)/$*:$(IMAGE_TAG) $(STAGING_DIR)/$*
	gcloud docker -- push gcr.io/$(GCP_PROJECT)/$*:$(IMAGE_TAG)

# Generates the yaml deployment from template and applies it to cluster.
deploy-%:	push-% gen-yaml-%
	kubectl apply -f $(GEN_YAML_DIR)/$*.yaml

deploy-common-objects:
	$(TOP_DIR)/scripts/deploy-common-objects.sh

# TODO(b/73013835): Remove deployment logic from Makefile.
deploy-resourcequota-admission-controller:	push-resourcequota-admission-controller gen-yaml-resourcequota-admission-controller
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
	kubectl delete ValidatingWebhookConfiguration stolos-resource-quota --ignore-not-found
	kubectl apply -f $(GEN_YAML_DIR)/resourcequota-admission-controller.yaml

all-deploy: $(addprefix deploy-, $(ALL_COMPONENTS))

