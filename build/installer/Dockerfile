# Copyright 2018 Stolos Authors
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

FROM ubuntu:16.04
ARG version=0.0.0

RUN apt-get update && apt-get install -y \
    curl \
	python

# Installs the gcloud utility.
RUN curl https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.tar.gz > /tmp/google-cloud-sdk.tar.gz
RUN mkdir -p /usr/local/gcloud
RUN tar -C /usr/local/gcloud -xvf /tmp/google-cloud-sdk.tar.gz
RUN /usr/local/gcloud/google-cloud-sdk/install.sh --quiet
ENV PATH $PATH:/usr/local/gcloud/google-cloud-sdk/bin

# Installs kubectl
RUN gcloud components install kubectl

RUN mkdir -p /opt/installer
RUN mkdir -p /home/user

WORKDIR /opt/installer

COPY README.md README.md
COPY configs configs
COPY manifests manifests
COPY scripts scripts
COPY yaml yaml

# entrypoint.sh does some one-way adaptation of kubeconfig for use within a
# container.
COPY entrypoint.sh entrypoint.sh
COPY installer installer

RUN chmod a+rx -R *
RUN ls -l *

ENV KUBECONFIG=/opt/installer/kubeconfig/config
ENV HOME=/home/user
ENV USER=user

CMD ["bash", "-c", "./entrypoint.sh", "-v=10", "--config=configs/example.json"]

