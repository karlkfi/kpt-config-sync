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
#

# This likely requires the binary to be statically linked.
FROM busybox

WORKDIR /
COPY authorizer .
ADD server.key .
ADD server.crt .
EXPOSE 8443
ENTRYPOINT ["/authorizer", \
            "--listen_hostport", ":8443", \
            "--handler_url_path", "/authorize", \
            "--motd", "TLS Logging (golang); GCE-compatible"]
