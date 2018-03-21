/*
Copyright 2017 The Nomos Authors.

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

package cli

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestEnvProcessing(t *testing.T) {
	os.Setenv("KUBECTL_PLUGINS_DESCRIPTOR_NAME", "nomos")
	os.Setenv("KUBECTL_PLUGINS_DESCRIPTOR_SHORT_DESC", "short-desc")
	os.Setenv("KUBECTL_PLUGINS_DESCRIPTOR_LONG_DESC", "long-desc")
	os.Setenv("KUBECTL_PLUGINS_DESCRIPTOR_EXAMPLE", "example")
	os.Setenv("KUBECTL_PLUGINS_DESCRIPTOR_COMMAND", "./kubectl-nomos")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_ALSOLOGTOSTDERR", "false")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_AS", "foo")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_AS_GROUP", "[]")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CACHE_DIR", "/usr/local/google/home/briankennedy/.kube/http-cache")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CERTIFICATE_AUTHORITY", "")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CLIENT_CERTIFICATE", "")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CLIENT_KEY", "")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CLUSTER", "")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CONTEXT", "")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_INSECURE_SKIP_TLS_VERIFY", "false")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_KUBECONFIG", "")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_LOG_BACKTRACE_AT", ":0")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_LOG_DIR", "")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_LOG_FLUSH_FREQUENCY", "5s")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_LOGTOSTDERR", "true")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_MATCH_SERVER_VERSION", "false")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_NAMESPACE", "")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_PASSWORD", "")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_REQUEST_TIMEOUT", "0")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_SERVER", "")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_STDERRTHRESHOLD", "2")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_TOKEN", "")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_USER", "")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_USERNAME", "")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_V", "0")
	os.Setenv("KUBECTL_PLUGINS_GLOBAL_FLAG_VMODULE", "")
	os.Setenv("KUBECTL_PLUGINS_LOCAL_FLAG_DEMO", "")
	os.Setenv("KUBECTL_PLUGINS_LOCAL_FLAG_HELP", "false")
	os.Setenv("KUBECTL_PLUGINS_LOCAL_FLAG_YEP", "nope")
	os.Setenv("KUBECTL_PLUGINS_CURRENT_NAMESPACE", "default")

	ctx := newCommandContext()
	ctx.initEnvironment()

	expectedCtx := &CommandContext{
		GlobalFlags: map[string]string{
			"alsologtostderr":          "false",
			"as_group":                 "[]",
			"cache_dir":                "/usr/local/google/home/briankennedy/.kube/http-cache",
			"certificate_authority":    "",
			"client_certificate":       "",
			"client_key":               "",
			"cluster":                  "",
			"context":                  "",
			"insecure_skip_tls_verify": "false",
			"kubeconfig":               "",
			"log_backtrace_at":         ":0",
			"log_dir":                  "",
			"log_flush_frequency":      "5s",
			"logtostderr":              "true",
			"match_server_version":     "false",
			"namespace":                "",
			"password":                 "",
			"request_timeout":          "0",
			"server":                   "",
			"stderrthreshold":          "2",
			"token":                    "",
			"user":                     "",
			"username":                 "",
			"v":                        "0",
			"vmodule":                  "",
		},
		LocalFlags: map[string]string{
			"demo": "",
			"help": "false",
			"yep":  "nope",
		},
		Name:      "nomos",
		ShortDesc: "short-desc",
		LongDesc:  "long-desc",
		Example:   "example",
		Command:   "./kubectl-nomos",
		Namespace: "default",
		asUser:    "foo",
		Client:    nil,
	}

	if !reflect.DeepEqual(expectedCtx, ctx) {
		fmt.Printf("%#v\n", ctx)
		t.Errorf("Unexpected state after enviornment processing, expected:\n%s\ngot:\n%s\n",
			spew.Sdump(expectedCtx), spew.Sdump(ctx))
	}
}
