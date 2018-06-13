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

// Controller responsible for importing policies from Google Cloud kubernetespolicy API and
// materializing CRDs on the local cluster.
package main

import (
	"flag"
	"os"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/policyimporter/gcp"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
)

var (
	orgID      = flag.String("org-id", os.Getenv("ORG_ID"), "organization ID")
	apiAddress = flag.String("policy-api-address", os.Getenv("POLICY_API_ADDRESS"), "Kubernetes Policy API address")
	credsFile  = flag.String("gcp-credentials-file", os.Getenv("GOOGLE_GCP_CREDENTIALS_FILE"), "the gcp service account credentials file to use to open the connection")
	caFile     = flag.String("ca-file", os.Getenv("GCP_POLICY_IMPORTER_CA_FILE"), "the root CA certificate file to use in place of the system one, e.g. for testing")
)

func main() {
	flag.Parse()
	log.Setup()

	if *orgID == "" {
		glog.Fatal("-org-id must be specified")
	}
	if *apiAddress == "" {
		glog.Fatal("-policy-api-address must be specified")
	}

	config, err := restconfig.NewRestConfig()
	if err != nil {
		glog.Fatalf("Failed to create rest config: %v", err)
	}

	client, err := meta.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	go service.ServeMetrics()

	stopChan := make(chan struct{})
	c := gcp.NewController(*orgID, *apiAddress, *credsFile, *caFile, client, stopChan)
	go service.WaitForShutdownSignalCb(stopChan)
	if err := c.Run(); err != nil {
		glog.Fatalf("Failure running controller: %v", err)
	}

	glog.Info("Exiting")
}
