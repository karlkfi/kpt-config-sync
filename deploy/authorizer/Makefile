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

# The fixed Cluster IP address that is currently used for the minikube authorzer
# deployment.  For the time being this must be an IP address, and not a FQDN.
MINIKUBE_AUTHORIZER_CLUSTER_IP_ADDRESS := 10.0.0.112


### make clean

.PHONY: clean
clean:
	rm -f Dockerfile $(PACKAGE)
	rm -f *.crt *.key *.pem *.csr *.tmp minikube/*.yaml
	rm -f gce/*.yaml

.PHONY: clean-docker
clean-docker: clean-docker-$(DEPLOYMENT_TYPE)

.PHONY: clean-docker-gce
clean-docker-gce:
	docker rmi $(PACKAGE):$(IMAGE_TAG)

.PHONY: clean-docker-minikube
clean-docker-minikube:
	eval $(MINIKUBE_DOCKER_ENV)
	docker rmi $(PACKAGE):$(IMAGE_TAG)

.NOTPARALLEL: server.key server.csr server.pem server.crt ca.crt
server.key server.csr server.pem server.crt: minikube/gencert.sh
	AUTHORIZER_CLUSTER_IP_ADDRESS=$(MINIKUBE_AUTHORIZER_CLUSTER_IP_ADDRESS) \
		./minikube/gencert.sh

$(HOME)/.minikube/addons/webhook.kubeconfig: \
  minikube/init.sh \
  server.key \
  server.crt
	WEBHOOK_ADDRESS=$(MINIKUBE_AUTHORIZER_CLUSTER_IP_ADDRESS) \
        	minikube/init.sh

.PHONY: minikube_start
minikube_start: $(HOME)/.minikube/addons/webhook.kubeconfig minikube/start.sh
	minikube/start.sh

.PHONY: minikube_stop
minikube_stop:
	minikube stop

Dockerfile: Dockerfile.in
	m4 -DBUILD_DIR=$(GO_BIN) \
	   -DBINARY_NAME=$(PACKAGE) \
	   -DGEN_NOTE="Generated file, do not edit." < $^ > $@

### make build-docker ...

# Only invoked in case the binary has not been built by top-level Makefile.
$(GO_BIN)/$(PACKAGE):
	CGO_ENABLED=0
	go install -installsuffix "static" $(REPO)/cmd/$(PACKAGE)

$(PACKAGE): $(GO_BIN)/$(PACKAGE)
	cp $(GO_BIN)/$(PACKAGE) $@

.PHONY: build-docker
build-docker: Dockerfile $(PACKAGE) server.key
	docker build -t $(PACKAGE):$(IMAGE_TAG) .

### make push-docker ...

.PHONY: push-docker
push-docker: build-docker push-docker-$(DEPLOYMENT_TYPE)

.PHONY: push-docker
push-docker-gce: build-docker
	docker tag $(PACKAGE):$(IMAGE_TAG) \
	  gcr.io/$(GCP_PROJECT)/$(PACKAGE):$(IMAGE_TAG)
	gcloud docker -- \
	  push gcr.io/$(GCP_PROJECT)/$(PACKAGE):$(IMAGE_TAG)

.PHONY: push-docker-minikube
push-docker-minikube: Dockerfile $(PACKAGE) server.key
	eval $(MINIKUBE_DOCKER_ENV)
	docker build -t $(PACKAGE):$(IMAGE_TAG) .

# Invoked if $(DEPLOYMENT_TYPE) is not found.
push-docker-%:
	$(error Deployment unknown: DEPLOYMENT_TYPE=$(DEPLOYMENT_TYPE))


.PHONY: deploy
deploy: deploy-$(DEPLOYMENT_TYPE)

.PHONY: deploy-minikube
deploy-minikube: minikube_start

gce/%.yaml: gce/%.yaml.in
	m4 -DGCP_PROJECT=$(GCP_PROJECT) \
	   -DPACKAGE=$(PACKAGE) \
	   -DCLUSTER_IP=$(MINIKUBE_AUTHORIZER_CLUSTER_IP_ADDRESS) \
	   -DIMAGE_TAG=$(IMAGE_TAG) \
	   -DGEN_NOTE="Generated file, do not edit." < $^ > $@

# Requires:
# KUBERNETES_VERSION
deploy-gce: gce/authorizer_deploy.yaml gce/authz.yaml

.PHONY: undeploy
undeploy: undeploy-$(DEPLOYMENT_TYPE)

.PHONY: undeploy-minikube
undeploy-minikube: minikube_stop

.PHONY: undeploy-gce
undeploy-gce:
	@echo "*** TODO(fmil): Implement target: $@"



