/*
Copyright 2017 The Kubernetes Authors.
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

// Scaffolding of a Kubernetes webhook authorizer.
package main

import (
	"encoding/json"
	"flag"
	"time"

	"github.com/coreos/go-systemd/daemon"
	"github.com/golang/glog"
	"github.com/google/stolos/pkg/authorizer"
	"github.com/google/stolos/pkg/client/informers/externalversions"
	meta "github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/service"
	"io/ioutil"
	authz "k8s.io/api/authorization/v1beta1"
	"k8s.io/client-go/rest"
	"net/http"
)

var (
	notifySystemd = flag.Bool(
		"notify_systemd", false,
		"Whether to notify systemd that the daemon is ready to serve. "+
			"Used if the service is ran from systemd, as opposed from a pod.")
	motd           = flag.String("motd", "", "This message is printed first.")
	clientCertFile = flag.String(
		"client_cert", "", "The client certificate file.")
)

// Responder writes a basic message out.
// See "Request Payloads" at:
// https://kubernetes-v1-4.github.io/docs/admin/authorization for details.
func Responder(a *authorizer.Authorizer) service.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		var body []byte
		if req.Body != nil {
			if data, err := ioutil.ReadAll(req.Body); err == nil {
				body = data
			}
		}

		// Run the type sanity checks: expect the correct content type,
		// then try to deserialize, and bomb out on invalid TypeMeta.
		contentType := req.Header.Get("Content-Type")
		if contentType != "application/json" {
			glog.Errorf(
				"contentType='%v', expect application/json",
				contentType)
			writer.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
		var reviewRequest authz.SubjectAccessReview
		err := json.Unmarshal(body, &reviewRequest)
		if err != nil {
			glog.Errorf(
				"Could not unmarshal as request spec: %v",
				body)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		// TODO(filmil): Is there a "canonical" way to check this?
		if reviewRequest.TypeMeta != authorizer.TypeMeta {
			glog.Errorf("Invalid TypeMeta: %v",
				reviewRequest.TypeMeta)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		// Process the request: call authorizer here.
		glog.V(1).Infof("Request: %+v", string(body))
		resp, err := json.Marshal(authz.SubjectAccessReview{
			TypeMeta: authorizer.TypeMeta,
			Status:   *a.Authorize(&reviewRequest.Spec),
		})
		if err != nil {
			glog.Errorf("While marshalling response: %v", err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		// If we survived up to here, write the response out and done.
		glog.V(2).Infof("Response: %+v", string(resp))
		writer.Write(resp)
	}
}

// ServeFunc returns the serving function for this server.  Use for testing.
func ServeFunc(a *authorizer.Authorizer) service.HandlerFunc {
	return service.WithStrictTransport(
		service.WithRequestLogging(
			service.NoCache(
				Responder(a))))
}

// must strips the error from a two-valued result.  Use type assertion to
// restore the original type of 'i'.
func must(i interface{}, err error) interface{} {
	if err != nil {
		glog.Fatalf("Error occurred: %v", err)
	}
	return i
}

// maybeNotifySystemd notifies the monitor daemon that we're ready to start
// serving.  But only if the daemon is actually on the other side, since the
// notification writes into a Unix socket under the hood.
func maybeNotifySystemd() {
	if *notifySystemd {
		daemon.SdNotify( /*unsetEnvironment=*/ false, "READY=1")
	}
}

func main() {
	flag.Parse()
	flag.Set("logtostderr", "true")

	glog.Infof("Motd: %v", *motd)
	config := must(rest.InClusterConfig()).(*rest.Config)
	client := must(meta.NewForConfig(config)).(*meta.Client)
	factory := externalversions.NewSharedInformerFactory(
		client.PolicyHierarchy(), time.Minute,
	)
	factory.Start(nil)

	var clientCert []byte
	if *clientCertFile != "" {
		clientCert = must(ioutil.ReadFile(*clientCertFile)).([]byte)
	}
	srv := service.Server(
		ServeFunc(authorizer.New(
			factory.K8us().V1().PolicyNodes().Informer())), clientCert)
	factory.Start(nil)

	maybeNotifySystemd()
	err := srv.ListenAndServeTLS("", "")
	if err != nil {
		glog.Fatalf("ListenAndServe: %+v", err)
	}
}
