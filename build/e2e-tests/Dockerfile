# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Build intermediate image for tools
FROM golang:1.11-alpine AS build-base
RUN apk add bash ca-certificates git gcc g++ libc-dev

ENV TAP2JUNIT_RELEASE=v0.0.4
RUN go get github.com/filmil/tap2junit/...
WORKDIR /go/src/github.com/filmil/tap2junit
RUN git checkout ${TAP2JUNIT_RELEASE}

# Produce a binary tap2junit in /go/bin/...
RUN CGO_ENABLED=0 go install github.com/filmil/tap2junit/...

# Build intermediate image for gcloud / kubectl
FROM ubuntu:16.04 as gcloud-install

ENV PATH /opt/gcloud/google-cloud-sdk/bin:$PATH
RUN apt-get update && apt-get install -y curl python
RUN curl https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-210.0.0-linux-x86_64.tar.gz > /tmp/google-cloud-sdk.tar.gz
RUN mkdir -p /opt/gcloud
RUN tar -C /opt/gcloud -xf /tmp/google-cloud-sdk.tar.gz
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

ARG UNAME
ARG UID
ARG GID

# The existence of a valid UID, GID and UNAME (USER) is required for programs
# that expect a "regular" unix environment, for example 'git'.
RUN bash -c 'echo UID=${UID}; GID=${GID}; UNAME=${UNAME}'

# This is needed to get build going on any environment with a nonprivileged
# user.  Turns out that on go/prow, the build is ran as root, so no need for
# any such setup.  Hence is done conditionally.
RUN bash -c '[ $GID == 0 ] || groupadd -or -g $GID grp '
RUN bash -c '[ $UID == 0 ] || useradd -u $UID -G $GID $UNAME'

# Install kubectl 1.11
RUN curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - \
  && echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list \
  && apt-get update \
  && apt-get install -y kubectl=1.13.3-00

# Copy installed gcloud into image
COPY --from=gcloud-install /opt/gcloud/google-cloud-sdk /opt/gcloud/google-cloud-sdk
ENV PATH /opt/gcloud/google-cloud-sdk/bin:$PATH

COPY --from=build-base /go/bin/tap2junit /opt

ENV GIT_CONFIG_NOGLOBAL 1
ENV GIT_CONFIG_NOSYSTEM 1

ENV USER $UNAME
USER $UNAME

WORKDIR /opt/testing/e2e
ENTRYPOINT ["/opt/testing/e2e/setup.sh"]

