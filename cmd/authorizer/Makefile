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

.PHONY: build clean test docker_minikube docker_clean authorizer cert
.PHONY: debug_authorizer_start minikube_start minikube_stop

PREFIX := authorizer
TAG := experimental

DOCKER_ENV := $(shell minikube docker-env)

# The fixed Cluster IP address that is currently used for the minikube authorzer
# deployment.  For the time being this must be an IP address, and not a FQDN.
MINIKUBE_AUTHORIZER_CLUSTER_IP_ADDRESS := 10.0.0.112

test:
	go test .

build: authorizer

authorizer:
	go build .


# CGO_ENABLED=0 will build a statically linked binary, which allows us to
# build a Docker container from scratch and get a very tiny container.
# TODO(filmil): CGO_ENABLED=0 build is too slow.  See if it can be sped up.
.PHONY: authorizer_no_cgo
authorizer_no_cgo:
	CGO_ENABLED=0 go build .



clean:
	go clean .
	rm -f *.crt *.key *.pem *.csr *.tmp minikube/*.yaml

# Use this target to run the test authorizer on your machine, with a debug
# setup
debug_authorizer_start: authorizer server.key server.crt
	./authorizer --logtostderr --vmodule=main=2

# Build a docker container for the minikube docker repository.
docker_minikube: test authorizer_no_cgo server.key server.crt
	eval $(DOCKER_ENV) && \
	docker build -t $(PREFIX):$(TAG) .

docker_clean:
	eval $(DOCKER_ENV) && \
	docker rmi $(PREFIX):$(TAG)

.NOTPARALLEL: server.key server.csr server.pem server.crt ca.crt
server.key server.csr server.pem server.crt: minikube/gencert.sh
	AUTHORIZER_CLUSTER_IP_ADDRESS=$(MINIKUBE_AUTHORIZER_CLUSTER_IP_ADDRESS) \
		./minikube/gencert.sh


$(HOME)/.minikube/addons/webhook.kubeconfig: \
  authorizer_no_cgo \
  minikube/init.sh \
  server.key \
  server.crt
	WEBHOOK_ADDRESS=$(MINIKUBE_AUTHORIZER_CLUSTER_IP_ADDRESS) \
        	minikube/init.sh

minikube_start: $(HOME)/.minikube/addons/webhook.kubeconfig minikube/start.sh
	minikube/start.sh

minikube_stop:
	minikube stop

