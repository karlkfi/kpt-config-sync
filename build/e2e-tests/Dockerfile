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

ENV DEBIAN_FRONTEND=noninteractive

# Install system stuff
RUN apt-get update
RUN apt-get upgrade -y
RUN apt-get install -y \
  apt-transport-https \
  apt-utils \
  ca-certificates \
  curl \
  git \
  jq \
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
RUN groupadd -or -g $DOCKER_GID docker
RUN useradd -u $UID -G $DOCKER_GID $UNAME

# Install docker as per docker website instructions.
RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -
RUN add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu xenial stable"
RUN apt-get update
RUN apt-get install -y docker-ce

ENV GIT_CONFIG_NOGLOBAL 1
ENV GIT_CONFIG_NOSYSTEM 1

ENV USER $UNAME
USER $UNAME

WORKDIR /opt/testing/e2e

ENTRYPOINT ["/opt/testing/e2e/setup.sh"]
