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

FROM alpine

# https://github.com/GoogleCloudPlatform/google-cloud-go/issues/791#issuecomment-353689746
RUN apk --no-cache --update add ca-certificates

ADD gcp-policy-importer gcp-policy-importer
