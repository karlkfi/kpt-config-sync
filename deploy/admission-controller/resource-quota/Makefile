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

.PHONY: test build certs deploy restart register

GCP_PROJECT ?= $(shell gcloud config get-value project)
DOCKER_ENV ?= $(shell minikube docker-env)

TAG := experimental
IMAGE := gcr.io/$(GCP_PROJECT)/resource-quota:$(TAG)
REPO := github.com/google/stolos
PACKAGE_PATH := $(REPO)/cmd/admission-controller/resource-quota

test:
	go test $(PACKAGE_PATH)

build:
	CGO_ENABLED=0 go build $(PACKAGE_PATH)

certs:
	./gencert.sh

deploy: test build
	eval $(DOCKER_ENV)
	docker build -t $(IMAGE) .
	gcloud docker -- push $(IMAGE)
	m4 -DIMAGE_NAME=$(IMAGE) < pod-template.m4 > pod.yaml
	./start_pod.sh
	rm pod.yaml

restart:
	m4 -DIMAGE_NAME=$(IMAGE) < pod-template.m4 > pod.yaml
	./start_pod.sh
	rm pod.yaml

register:
	./register.sh
