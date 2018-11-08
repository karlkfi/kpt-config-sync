# Copyright 2017 Nomos Authors
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

FROM k8s.gcr.io/debian-base:0.4.0

# https://github.com/GoogleCloudPlatform/google-cloud-go/issues/791#issuecomment-353689746
RUN apt-get update && apt-get install -y \
  ca-certificates

RUN mkdir -p /opt/nomos/bin
ENV PATH="/opt/nomos/bin:${PATH}"
WORKDIR /opt/nomos/bin

# Nomos binaries.
COPY gcp-policy-importer gcp-policy-importer
COPY git-policy-importer git-policy-importer
COPY monitor monitor
COPY nomosvet nomosvet
COPY resourcequota-admission-controller resourcequota-admission-controller
COPY syncer syncer

# License file required for on-prem release.
COPY LICENSE LICENSE

# Required by admission controllers.
EXPOSE 443

CMD echo -e "Need to specify a Nomos binary:\n\n$(ls .)" && exit 1
