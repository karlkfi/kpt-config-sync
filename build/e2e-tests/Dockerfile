# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Build intermediate image for gcloud / kubectl
FROM ubuntu:16.04 as gcloud-install

ENV PATH /opt/gcloud/google-cloud-sdk/bin:$PATH
RUN apt-get update && apt-get install -y curl python
RUN curl https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-210.0.0-linux-x86_64.tar.gz > /tmp/google-cloud-sdk.tar.gz
RUN mkdir -p /opt/gcloud
RUN tar -C /opt/gcloud -xvf /tmp/google-cloud-sdk.tar.gz
RUN /opt/gcloud/google-cloud-sdk/install.sh --quiet
RUN gcloud components install alpha beta

# Build e2e image
FROM ubuntu:16.04 as e2e-base

ENV DEBIAN_FRONTEND=noninteractive

# Install system stuff
RUN apt-get update \
  && apt-get upgrade -y \
  && apt-get install -y \
  apt-transport-https \
  apt-utils \
  ca-certificates \
  curl \
  git \
  jq \
  libtap-formatter-junit-perl \
  python \
  software-properties-common

# Set up user (this has to go before docker installation so the GID can match host system)
# This does the following:
#   Creates group docker with same GID as host system
#   Creates user with same UID as host system in docker and primarygroup group
ARG UNAME
ARG UID
ARG GID
ARG DOCKER_GID

# Install kubectl 1.11
RUN curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - \
  && echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list \
  && apt-get update \
  && apt-get install -y kubectl=1.11.1-00

# Install docker as per docker website instructions.
RUN bash -c 'echo "DOCKER_GID=${DOCKER_GID}; UID=${UID}; GID=${GID}; UNAME=${UNAME}"'
#RUN groupadd -or -g $DOCKER_GID docker \
  #&& useradd -u $UID -G $DOCKER_GID $UNAME \
  #&& curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add - \
  #&& add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu xenial stable" \
  #&& apt-get update \
  #&& apt-get install -y docker-ce

# Copy installed gcloud into image
COPY --from=gcloud-install /opt/gcloud/google-cloud-sdk /opt/gcloud/google-cloud-sdk
ENV PATH /opt/gcloud/google-cloud-sdk/bin:$PATH

ENV GIT_CONFIG_NOGLOBAL 1
ENV GIT_CONFIG_NOSYSTEM 1

ENV USER $UNAME
USER $UNAME

WORKDIR /opt/testing/e2e

ENTRYPOINT ["/opt/testing/e2e/setup.sh"]

