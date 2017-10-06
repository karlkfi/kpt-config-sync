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

MINIKUBE_DOCKER_ENV ?= $(shell minikube docker-env)
PACKAGE := authorizer
IMAGE_TAG ?= test
GO_BIN ?= undefined_go_bin_directory
GCP_PROJECT ?= undefined_gcp_project
DEPLOYMENT_TYPE ?= gce
KUBERNETES_VERSION ?= 1.8.0


### make clean

.PHONY: clean
clean:
	@echo "*** TODO(fmil): Implement target: $@"

.PHONY: clean-docker
clean-docker: clean-docker-$(DEPLOYMENT_TYPE)

.PHONY: clean-docker-gce
clean-docker-gce:
	@echo "*** TODO(fmil): Implement target: $@"

.PHONY: clean-docker-minikube
clean-docker-minikube:
	@echo "*** TODO(fmil): Implement target: $@"

### make build-docker ...

.PHONY: build-docker
build-docker:
	@echo "*** TODO(fmil): Implement target: $@"


### make push-docker ...

.PHONY: push-docker
push-docker: build-docker push-docker-$(DEPLOYMENT_TYPE)

.PHONY: push-docker
push-docker-gce: build-docker
	@echo "*** TODO(fmil): Implement target: $@"

.PHONY: push-docker-minikube
push-docker-minikube: build-docker
	@echo "*** TODO(fmil): Implement target: $@"

# Invoked if $(DEPLOYMENT_TYPE) is not found.
push-docker-%:
	$(error Deployment unknown: DEPLOYMENT_TYPE=$(DEPLOYMENT_TYPE))


.PHONY: deploy
deploy: deploy-$(DEPLOYMENT_TYPE)

.PHONY: deploy-minikube
deploy-minikube:
	@echo "*** TODO(fmil): Implement target: $@"


# Requires:
# KUBERNETES_VERSION

deploy-gce:
	@echo "*** TODO(fmil): Implement target: $@"

.PHONY: undeploy
undeploy: undeploy-$(DEPLOYMENT_TYPE)
	@echo "*** TODO(fmil): Implement target: $@"

.PHONY: undeploy-gce
undeploy-gce:
	@echo "*** TODO(fmil): Implement target: $@"

.PHONY: undeploy-minikube
undeploy-minikube:
	@echo "*** TODO(fmil): Implement target: $@"

