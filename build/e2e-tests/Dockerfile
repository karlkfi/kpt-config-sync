# Copyright 2018 Nomos Authors
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

RUN apt-get update && apt-get install -y \
    curl \
	dialog \
	python

# Installs the gcloud utility.
RUN curl https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.tar.gz > /tmp/google-cloud-sdk.tar.gz \
    && mkdir -p /usr/local/gcloud \
    && tar -C /usr/local/gcloud -xvf /tmp/google-cloud-sdk.tar.gz \
    && /usr/local/gcloud/google-cloud-sdk/install.sh --quiet
ENV PATH $PATH:/usr/local/gcloud/google-cloud-sdk/bin

# Installs kubectl
RUN gcloud components install kubectl

RUN mkdir -p /opt/installer/kubeconfig
RUN mkdir -p /home/user

WORKDIR /opt/installer

COPY examples examples
COPY manifests manifests
COPY scripts scripts
COPY yaml yaml

COPY installer installer
COPY install-config.yaml install-config.yaml
COPY test-syncer-change-rolebinding-backend.yaml test-syncer-change-rolebinding-backend.yaml
COPY e2e-entrypoint.sh e2e-entrypoint.sh

RUN chmod a+rx -R *
RUN ls -l *

ENV KUBECONFIG=/opt/installer/kubeconfig/config
ENV HOME=/home/user
ENV USER=user
ENV PATH=$PATH:/opt/installer

ARG INSTALLER_VERSION="unset"
ENV INSTALLER_VERSION=$INSTALLER_VERSION

# TODO(ekitson): Make the installer part of the image when we have a static kubeconfig for e2e
#RUN installer \
#    --config="install-config.yaml" \
#    --log_dir=/tmp \
#    --use_current_context=true \
#    --vmodule=main=10,configgen=10,kubectl=10,installer=10,exec=10

ENTRYPOINT ["./e2e-entrypoint.sh", "--alsologtostderr"]
