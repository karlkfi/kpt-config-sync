# Included by Makefile.
# Rules related to building bespin and docker images.
#
# NOTE: Bespin binaries must contain the string "bespin" in its name -
#       that's how we exclude Bespin binaries from being uploaded to GCS
#       as part of Nomos release process.

###################################
# Bespin config
###################################

# Bespin artifacts.
BESPIN_OUTPUT_DIR := $(OUTPUT_DIR)/bespin

# Bespin build binaries.
BESPIN_BIN_DIR := $(BESPIN_OUTPUT_DIR)/bin

# Bespin build related files.
BESPIN_BUILD_DIR := $(TOP_DIR)/build/bespin

# Bespin manifests.
BESPIN_CONFIG_DIR := $(TOP_DIR)/manifests/bespin/config

# Image URLs for bespin image targets.
BESPIN_CONTROLLERS_MANAGER_IMG ?= gcr.io/${GCR_PREFIX}/bespin-controllers-manager:$(IMAGE_TAG)

###################################
# Bespin build rules
###################################

bespin-all: bespin-test bespin-controllers-manager

# Run bespin related tests
bespin-test:
	go test -short ./pkg/bespin-controllers/...
	go test -short ./pkg/api/policyascode/v1/...

# Build bespin controllers manager binary
# TODO(b/119874295): build in a container
bespin-controllers-manager:
	go build -o $(BESPIN_BIN_DIR)/bespin-controllers-manager ./cmd/bespin-controllers-manager/main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
bespin-run-manager-local: bespin-terraform
	go run ./cmd/bespin-controllers-manager/main.go  --tf_binary_path="${BESPIN_OUTPUT_DIR}/terraform-bundle/terraform" --tf_plugin_dir="${BESPIN_OUTPUT_DIR}/terraform-bundle" --alsologtostderr=true --v=10

# Install Bespin CRDs into a cluster
bespin-install-crds:
	kubectl apply -f ${BESPIN_CONFIG_DIR}/crds

# Apply RBAC roles and role bindings.
bespin-install-rbac:
	kubectl apply -f ${BESPIN_CONFIG_DIR}/rbac

# Deploy bespin controllers manager separately.
bespin-deploy: bespin-install-crds bespin-install-rbac
	kubectl delete StatefulSet --all -n bespin-system
	kubectl apply -f ${BESPIN_CONFIG_DIR}/manager

# Generate the bespin controllers manager YAML and put it to staging.
gen-yaml-bespin-controllers-manager:
	@echo "+++ Generating yaml bespin-controllers-manager"
	@IMAGE_NAME=$(BESPIN_CONTROLLERS_MANAGER_IMG) envsubst < \
			$(BESPIN_CONFIG_DIR)/manager/bespin-controllers-manager.yaml > \
			$(STAGING_DIR)/installer/manifests/deployment/bespin-controllers-manager.yaml

bespin-installer-staging: installer-staging \
		gen-yaml-bespin-controllers-manager \
		bespin-image-push
	@echo "+++ bespin-installer-staging"
	@cp $(BESPIN_CONFIG_DIR)/crds/*.yaml $(STAGING_DIR)/installer/manifests
	@cp $(BESPIN_CONFIG_DIR)/rbac/*.yaml $(STAGING_DIR)/installer/manifests

bespin-deploy-operator-user: bespin-image-push deploy-operator-user
	@echo "+++ Finished deploying bespin operator"

# Build terraform binary
bespin-terraform:
	@echo "+++ Downloading latest terraform and building terraform bundle"
	mkdir -p ${GOPATH}/src/github.com/hashicorp
	# Terraform needs to be pinned to a specific version. Nomos won't want
	# a large dependency in their vendor tree. This will be easier when Bespin
	# gets its own repository.
	git -C ${GOPATH}/src/github.com/hashicorp clone https://github.com/hashicorp/terraform.git -b v0.11.1 || true
	go install github.com/hashicorp/terraform/tools/terraform-bundle
	cd $(BESPIN_OUTPUT_DIR); \
	$(GOPATH)/bin/terraform-bundle package $(BESPIN_BUILD_DIR)/terraform-bundle.hcl && \
	mkdir -p terraform-bundle && unzip -o -d terraform-bundle terraform_*.zip && \
	rm -rf terraform_*.zip

###################################
# Docker images
###################################

# Build the bespin image.
bespin-image-build: bespin-controllers-manager bespin-terraform
	docker build . -t ${BESPIN_CONTROLLERS_MANAGER_IMG} -f build/bespin/Dockerfile

# Push the bespin image to gcr.
bespin-image-push: bespin-image-build
	@echo "+++ Pushing bespin-image-push to ${BESPIN_CONTROLLERS_MANAGER_IMG}..."
	docker push ${BESPIN_CONTROLLERS_MANAGER_IMG}
