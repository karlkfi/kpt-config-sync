# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ARG VERSION=latest
ARG GCP_PROJECT=stolos-dev
FROM gcr.io/$GCP_PROJECT/installer:$VERSION

ARG UNAME=testuser
ARG UID=1000
ARG GID=1000

ENV GIT_CONFIG_NOGLOBAL 1
ENV GIT_CONFIG_NOSYSTEM 1
RUN apt-get update && apt-get install -y git
RUN groupadd -g $GID $UNAME
RUN useradd -u $UID -g $UNAME $UNAME

ENTRYPOINT ["/opt/testing/e2e-bats-setup.sh"]
