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

.PHONY: build clean test docker_minikube docker_clean webhook_authorizer

PREFIX := webhook_authorizer
TAG := experimental

DOCKER_ENV := $(shell minikube docker-env)

all: docker_minikube


build: webhook_authorizer

# CGO_ENABLED=0 will build a statically linked binary, which allows us to
# build a Docker container from scratch and get a very tiny container.
# TODO(filmil): CGO_ENABLED=0 build is too slow.  See if it can be sped up.
webhook_authorizer:
	CGO_ENABLED=0 go build .


test: build
	go test .


clean:
	go clean .

# Build a docker container for the minikube docker repository.
docker_minikube: build test
	eval $(DOCKER_ENV) && docker build -t $(PREFIX):$(TAG) .

docker_clean:
	eval $(DOCKER_ENV) && docker rmi $(PREFIX):$(TAG)
