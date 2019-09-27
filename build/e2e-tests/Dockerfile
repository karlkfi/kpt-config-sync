# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# *** WARNING ***
# This image is built once and reused on repeated test runs.  If you make a
# change here, you *must* rebuild the image and push it to the Nomos container
# registry so that new test runs see the updated image.  This is not ideal but
# we don't have a better approach at the moment.
#
# To rebuild and push the image, run:
#     make push-e2e-tests

# Build intermediate image for tools
# When adding helper binaries, ensure to build from a specific commit ID, rather
# than a tag or a branch, so that rebuilds are reproducible. See examples below
# on how.
FROM golang:1.12.5-alpine AS build-base
RUN apk add bash ca-certificates coreutils git gcc g++ libc-dev

# We don't use or support modules for the time being.
ENV GO111MODULE=off
# Build static binaries only.
ENV CGO_ENABLED=0

# tap2junit converts TAP testing output to jUnit
# v0.0.4
ARG TAP2JUNIT_RELEASE="afa095c33668ed9abcaeaee5cf0f27f67e96908c"
ARG TAP2JUNIT_REPO="github.com/filmil/tap2junit"
RUN go get ${TAP2JUNIT_REPO}/...
WORKDIR /go/src/${TAP2JUNIT_REPO}
RUN git checkout ${TAP2JUNIT_RELEASE} && go install ${TAP2JUNIT_REPO}/...

# gotopt2 parses command line options
# v0.1.2
ARG GOTOPT2_RELEASE="484e7f416f00d80d55062b7bd5332fe24659694d"
ARG GOTOPT2_REPO="github.com/filmil/gotopt2/..."
RUN go get ${GOTOPT2_REPO}/...
WORKDIR /go/src/${GOTOPT2_REPO}
RUN git checkout ${GOTOPT2_RELEASE} && go install ${GOTOPT2_REPO}/...

# kind can run kubernetes clusters in docker
# v0.2.1
# TODO(juliankatz): Reevaluate the use of GO111MODULE.  It conflicts with declaration at the top.
RUN GO111MODULE="on" go get sigs.k8s.io/kind@v0.2.1

# Build intermediate image for gcloud / kubectl
FROM ubuntu:18.04 as gcloud-install

ENV PATH /opt/gcloud/google-cloud-sdk/bin:$PATH
RUN apt-get update && apt-get install -y curl python
ARG DL_GOOGLE_COM="https://dl.google.com/dl/cloudsdk/channels/rapid/downloads"
RUN curl ${DL_GOOGLE_COM}/google-cloud-sdk-255.0.0-linux-x86_64.tar.gz \
	> /tmp/google-cloud-sdk.tar.gz
# Verify SHA256 sum for the downloaded archive for reproducibility.
RUN echo "18fcbc81b3b095ff5ef92fd41286a045f782c18d99a976c0621140a8fde3fbad /tmp/google-cloud-sdk.tar.gz" | \
		sha256sum --check -
RUN mkdir -p /opt/gcloud && \
    tar -C /opt/gcloud -xf /tmp/google-cloud-sdk.tar.gz && \
    /opt/gcloud/google-cloud-sdk/install.sh --quiet && \
    gcloud components install alpha beta

# Build e2e image
FROM ubuntu:18.04 as e2e-base

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

# Add Tini
ENV TINI_VERSION v0.18.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini /tini
RUN (echo "12d20136605531b09a2c2dac02ccee85e1b874eb322ef6baf7561cd93f93c855 /tini" | sha256sum --check - ) && chmod +x /tini

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

# Install a specific version of kubectl
RUN curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - \
  && echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list \
  && apt-get update \
  && apt-get install -y kubectl=1.15.1-00

# Copy installed gcloud into image
COPY --from=gcloud-install /opt/gcloud/google-cloud-sdk /opt/gcloud/google-cloud-sdk
ENV PATH /opt/gcloud/google-cloud-sdk/bin:$PATH

COPY --from=build-base /go/bin/* /opt/

ENV GIT_CONFIG_NOGLOBAL 1
ENV GIT_CONFIG_NOSYSTEM 1

ENV USER $UNAME
USER $UNAME

WORKDIR /opt/testing/e2e

# tini (especially with -g) does correct handling of signals (like Ctrl+C) that you don't
# get by running a shell script as the entrypoint. If this were removed, Ctrl+C wouldn't
# work.
ENTRYPOINT ["/tini", "-g", "--"]
