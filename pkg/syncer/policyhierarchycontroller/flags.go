/*
Copyright 2018 The Nomos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package policyhierarchycontroller

import (
	"flag"
	"strings"
)

var hardSyncHelp = strings.Trim(`
Hard sync will cause the syncer to force the namespace list and policies to
exactly match the source of truth.  Note that this could potentially delete
running workloads if used improperly, so this should only be set for running on
a new cluster, or if you know exactly what you are doing.
`, "\n")

var strictNamespaceSync = flag.Bool("hardSync", false, hardSyncHelp)
