# Included by Makefile.
# Rules related to building bespin and docker images.
#
# NOTE: Bespin binaries must contain the string "bespin" in its name -
#       that's how we exclude Bespin binaries from being uploaded to GCS
#       as part of Nomos release process.

###################################
# Bespin config
###################################

TOP_DIR := $(shell pwd)
OUTPUT_DIR := $(TOP_DIR)/.output
BIN_DIR := $(OUTPUT_DIR)/bin
BUILD_DIR := $(TOP_DIR)/build/
CONFIG_DIR := $(TOP_DIR)/manifests/bespin/config

# Image URLs for Bespin image targets.
MANAGER_IMG ?= gcr.io/${GCR_PREFIX}/controllers-manager:$(IMAGE_TAG)

###################################
# Bespin build rules
###################################

all: test controllers-manager

# Run bespin related tests
test:
	go test -short ./pkg/controllers/...
	go test -short ./pkg/api/bespin/v1/...

# Build bespin controllers manager binary
# TODO(b/119874295): build in a container
controllers-manager:
	go build -o $(BIN_DIR)/bespin-controllers-manager ./cmd/controllers-manager/main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run-manager-local: terraform
	go run ./cmd/bespin-controllers-manager/main.go  --tf_binary_path="${OUTPUT_DIR}/terraform-bundle/terraform" --tf_plugin_dir="${OUTPUT_DIR}/terraform-bundle" --alsologtostderr=true --v=10

# Install Bespin CRDs into a cluster
install-crds:
	kubectl apply -f ${CONFIG_DIR}/crds

# Apply RBAC roles and role bindings.
install-rbac:
	kubectl apply -f ${CONFIG_DIR}/rbac

# Deploy bespin controllers manager separately.
deploy: install-crds install-rbac
	kubectl delete StatefulSet --all -n bespin-system
	kubectl apply -f ${CONFIG_DIR}/manager

# Generate the bespin controllers manager YAML and put it to staging.
gen-yaml-controllers-manager:
	@echo "+++ Generating yaml controllers-manager"
	@IMAGE_NAME=$(MANAGER_IMG) envsubst < \
			$(CONFIG_DIR)/manager/controllers-manager.yaml > \
			$(STAGING_DIR)/installer/manifests/deployment/bespin-controllers-manager.yaml

installer-staging: installer-staging \
		gen-yaml-controllers-manager \
		image-push
	@echo "+++ installer-staging"
	@cp $(CONFIG_DIR)/crds/*.yaml $(STAGING_DIR)/installer/manifests
	@cp $(CONFIG_DIR)/rbac/*.yaml $(STAGING_DIR)/installer/manifests

# TODO(b/123758600): E2E is currently broken as this make rule depends on a Nomos one.
deploy-operator-user: image-push deploy-operator-user
	@echo "+++ Finished deploying bespin operator"

# Build terraform binary
terraform:
	@echo "+++ Downloading latest terraform and building terraform bundle"
	mkdir -p ${GOPATH}/src/github.com/hashicorp
	# Terraform needs to be pinned to a specific version. Nomos won't want
	# a large dependency in their vendor tree. This will be easier when Bespin
	# gets its own repository.
	git -C ${GOPATH}/src/github.com/hashicorp clone https://github.com/hashicorp/terraform.git -b v0.11.1 || true
	go install github.com/hashicorp/terraform/tools/terraform-bundle
	cd $(OUTPUT_DIR); \
	$(GOPATH)/bin/terraform-bundle package $(BUILD_DIR)/terraform-bundle.hcl && \
	mkdir -p terraform-bundle && unzip -o -d terraform-bundle terraform_*.zip && \
	rm -rf terraform_*.zip

###################################
# Docker images
###################################

# Build the bespin image.
image-build: controllers-manager terraform
	docker build . -t ${MANAGER_IMG} -f $(BUILD_DIR)/Dockerfile

# Push the bespin image to gcr.
image-push: image-build
	@echo "+++ Pushing image-push to ${MANAGER_IMG}..."
	docker push ${MANAGER_IMG}
