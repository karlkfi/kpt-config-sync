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

.PHONY: clean
clean: $(addprefix clean-,$(BINARIES))

clean-%:
	make -C $* clean


.PHONY: build-docker
build-docker: $(addprefix build-docker-,$(BINARIES))

build-docker-%:
	make -C $* build-docker


.PHONY: push-docker
push-docker: $(addprefix push-docker-,$(BINARIES))

push-docker-%:
	make -C $* push-docker

# Deploys all components to a target DEPLOYMENT_TYPE.
#
# Example:
# DEPLOYMENT_TYPE=minikube make deploy-authorizer

.PHONY: deploy
deploy: $(addprefix deploy-,$(BINARIES))

deploy-%:
	make -C $* deploy


# Revert the deployment of type DEPLOYMENT_TYPE
#
# Example:
# DEPLOYMENT_TYPE=minikube make undeploy-authorizer

.PHONY: deploy
undeploy: $(addprefix undeploy-,$(BINARIES))

undeploy-%:
	make -C $* undeploy

