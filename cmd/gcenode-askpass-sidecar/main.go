/*
  Note: cydu@ wrote this and was planning to maintain it at
  github.com/cydu-cloud/git-askpass-gce-node, but since there's little
  value in having it be open source we're incorporating it with his
  permission in the ACM code base.
*/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/golang/glog"
)

var port = flag.Int("port", 9102, "port to listen on")

type authToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int32  `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

func main() {
	flag.Parse()
	http.HandleFunc("/git_askpass", authServer)
	if err := http.ListenAndServe(":"+strconv.Itoa(*port), nil); err != nil {
		glog.Fatalf("HTTP ListenAndServe for gcenode-askpass-sidecar: %v", err)
	}
}

func authServer(w http.ResponseWriter, r *http.Request) {
	username, password, err := getGCENodeCredentials()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := fmt.Fprintf(w, "%v", err); err != nil {
			glog.Fatalf("Error writing HTTP header to report cred failure : %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintf(w, "username=%s\npassword=%s", username, password); err != nil {
		glog.Fatalf("Error writing auth information to HTTP header: %v", err)
	}
}

func getGCENodeCredentials() (string, string, error) {
	emailRequest := "instance/service-accounts/default/email"
	email, err := doMetadataGet(emailRequest)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch email, err: %v", err)
	}

	tokenRequest := "instance/service-accounts/default/token"
	token, err := doMetadataGet(tokenRequest)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch token, err: %v", err)
	}

	at := authToken{}
	if err = json.Unmarshal([]byte(token), &at); err != nil {
		return "", "", fmt.Errorf("unmarshal json failed: %s, err: %v", token, err)
	}
	return email, at.AccessToken, nil
}

func doMetadataGet(suffix string) (string, error) {
	var netClient = &http.Client{
		Timeout: time.Second * 1,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, _ := http.NewRequest("GET", "http://metadata/computeMetadata/v1/"+suffix, nil)
	req.Header.Add("Metadata-Flavor", "Google")
	resp, err := netClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch metadata %s, err: %v", suffix, err)
	}
	defer resp.Body.Close() // nolint: errcheck

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("invalid status code from metadata, resp: %v", resp)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response from metadata %s, err: %v", suffix, err)
	}
	return string(body), nil
}
