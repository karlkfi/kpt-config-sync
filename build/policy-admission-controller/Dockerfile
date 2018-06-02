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

FROM busybox
ADD policy-admission-controller policy-admission-controller
EXPOSE 8000
ENTRYPOINT ["/policy-admission-controller", \
        "--enable-mutual-tls=false", \
        "--server_key", "/etc/nomos/secrets/policy-admission-controller/server/tls.key", \
        "--server_cert", "/etc/nomos/secrets/policy-admission-controller/server/tls.crt", \
        "--ca-cert", "/etc/nomos/secrets/policy-admission-controller/ca/tls.crt"]

