package oidc

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	oidcclient "github.com/coreos/go-oidc"
	"github.com/davecgh/go-spew/spew"
	"github.com/filmil/k8s-oidc-helper/pkg/helper"
	"github.com/filmil/k8s-oidc-helper/pkg/oidc"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/kinds"
	oidcconfig "github.com/google/nomos/pkg/oidc/config"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
)

var (
	clusterConfig string
	cfg           oidc.Config
	// Set with flags
	issuerURL   string
	scope       string
	port        int
	openBrowser bool
	writeConfig bool
)

func init() {
	loginCmd := &cobra.Command{
		Use:     "login",
		Short:   "Logs in with some OIDC credentials",
		Run:     login,
		PreRunE: loginCheck,
	}
	loginCmd.Flags().StringVar(&clusterConfig, "cluster-config",
		os.Getenv("KUBECTL_PLUGINS_LOCAL_FLAG_CLUSTER_CONFIG"),
		"The cluster configuration generated by the cluster admin using the "+
			"'get-credentials' command")
	loginCmd.Flags().StringVar(&issuerURL, "issuer-url",
		"https://accounts.google.com", "The issuer URL.")
	loginCmd.Flags().StringVar(&scope, "scope",
		"openid+email+https://www.googleapis.com/auth/userinfo.groups",
		"The scopes to request; by default requests groups")
	loginCmd.Flags().IntVar(&port, "port",
		9789,
		"http://localhost:PORT is the redirect URI used for the OAuth flow.")
	loginCmd.Flags().BoolVar(&openBrowser, "open-browser", true,
		"If set, open a browser to log in, else print a URL for the user to open")
	loginCmd.Flags().BoolVar(&writeConfig,
		"write-config", true, "If set, writes kubectl config for theuser")

	rootCmd.AddCommand(loginCmd)
}

func loginCheck(_ *cobra.Command, _ []string) error {
	if clusterConfig == "" {
		return fmt.Errorf("value for flag --cluster-config=... is required")
	}
	return nil
}

// loginFlow absorbs all errors encountered while running its multiple
// methods.
type loginFlow struct {
	// Err stores last seen error.
	Err error
}

// login implements the "kubectl plugin oidc login" command.
func login(cmd *cobra.Command, _ []string) {
	var l loginFlow

	result, cid := l.LoadKubeConfig(clusterConfig)
	if l.Err != nil {
		// Without result and cid, there is no point in continuing from here.
		glog.Errorf("could not load file: %q", clusterConfig)
		os.Exit(2)
		return
	}
	clusterName := l.FirstCluster(result.Clusters)

	// Run the OIDC flow.
	cfg.ClientID = cid.Installed.ClientID
	cfg.ClientSecret = cid.Installed.ClientSecret

	// Prefix authInfoName with the cluster name, to allow multiple OIDC
	// identities to coexist.
	// For example, if cluster's name is "cluster-1", then clusterName will
	// be "oidc:cluster-1" and authInfoName will be "oidc:cluster-1:user@example.com"
	cfg.AuthInfoPrefix = fmt.Sprintf("%s:", clusterName)
	authInfoName := l.Run(cfg, issuerURL)

	// Make the new context be the current one.
	context := l.Context(clusterName, authInfoName)
	result.Contexts[clusterName] = context
	result.CurrentContext = clusterName

	// Write the result out.
	l.MergeKubeConfigs(result)

	if l.Err != nil {
		glog.Errorf("%s", l.Err)
		os.Exit(1)
	}
}

// LoadKubeConfig loads an extended configuration file from 'clusterConfig'.  Returns the
// full content of the configuration, and separately the client ID setting.
func (l *loginFlow) LoadKubeConfig(clusterConfig string) (*api.Config, *oidcconfig.ClientID) {
	if l.Err != nil {
		return nil, nil
	}
	r := l.Open(clusterConfig)
	b := l.ReadAll(r)
	json := l.ReadJSON(b)
	result := l.Decode(json)
	return l.Reparse(result)
}

// MergeKubeConfigs merges configuration 'result' with the current content of
// the user's kube config file.
func (l *loginFlow) MergeKubeConfigs(result *api.Config) {
	if l.Err != nil {
		return
	}
	tempKubeConfig := l.TempFile()
	tempName := tempKubeConfig.Name()
	defer func() {
		if err := os.Remove(tempName); err != nil {
			glog.Errorf("error while removing temp file: %q: %v", tempName, err)
		}
	}()

	l.WriteToFile(result, tempName)
	mergedConfig := l.mergeWithKubeConfig(tempName)
	// The embedded configuration is no longer needed, so remove it.
	delete(mergedConfig.Extensions, NomosOIDCExtensionKey)
	kubeConfigFile := l.kubeConfigName()
	l.WriteToFile(mergedConfig, kubeConfigFile)
}

// Context returns a new Context that joins the named cluster and user auth information.
// No checks are made, it is assumed that both supplied names are valid.
func (l *loginFlow) Context(clusterName, authInfoName string) *api.Context {
	if l.Err != nil {
		return nil
	}
	c := api.Context{
		AuthInfo: authInfoName,
		Cluster:  clusterName,
	}
	return &c
}

// FirstCluster returns the first cluster name from the supplied cluster map.
func (l *loginFlow) FirstCluster(cc map[string]*api.Cluster) string {
	if l.Err != nil {
		return ""
	}
	for name := range cc {
		return name
	}
	l.Err = fmt.Errorf("no clusters found in the config, expected exactly one")
	return ""
}

// WriteToFile writes the API configuration robustly into filename.
func (l *loginFlow) WriteToFile(result *api.Config, filename string) {
	if l.Err != nil {
		return
	}
	if result == nil {
		l.Err = fmt.Errorf("no config to write")
		return
	}
	if glog.V(4) {
		glog.Infof("api.Config: %v", spew.Sdump(result))
	}
	if err := clientcmd.WriteToFile(*result, filename); err != nil {
		l.Err = fmt.Errorf("while writing file: %q: %v", filename, err)
		return
	}
}

// Open opens the given file robustly.
func (l *loginFlow) Open(clusterConfig string) io.Reader {
	if l.Err != nil {
		return nil
	}
	r, err := os.Open(clusterConfig)
	if err != nil {
		l.Err = fmt.Errorf("while opening %q: %v", clusterConfig, err)
		return nil
	}
	return r
}

// ReadAll loads the cluster configuration bytes.
func (l *loginFlow) ReadAll(r io.Reader) []byte {
	if l.Err != nil {
		return nil
	}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		l.Err = fmt.Errorf("could not read %q: %v", clusterConfig, err)
		return nil
	}
	return b
}

// ReadJSON turns the YAML in b into JSON.
func (l *loginFlow) ReadJSON(b []byte) []byte {
	if l.Err != nil {
		return nil
	}
	json, err := yaml.YAMLToJSON(b)
	if err != nil {
		l.Err = fmt.Errorf("could not read as YAML: %v", err)
		return nil
	}
	if glog.V(7) {
		glog.Infof("json: %q", string(json))
	}
	return json
}

// Decode decodes a JSON byte array into Kubernetes configuration.
func (l *loginFlow) Decode(json []byte) *api.Config {
	if l.Err != nil {
		return nil
	}
	result := api.NewConfig()
	if err := runtime.DecodeInto(clientcmdlatest.Codec, json, result); err != nil {
		l.Err = fmt.Errorf("could not decode config %v", err)
		return nil
	}
	if glog.V(7) {
		glog.Infof("json: %+v", result)
	}
	return result
}

// Reparse does another round of parsing to turn the embedded unknown object
// into ClientID.  It seems as if this bit could be automatic, but that never
// really happens, and I haven't found examples of how to do it.
func (l *loginFlow) Reparse(result *api.Config) (*api.Config, *oidcconfig.ClientID) {
	if l.Err != nil {
		return nil, nil
	}
	raw := result.Extensions[NomosOIDCExtensionKey].(*runtime.Unknown)
	if glog.V(8) {
		glog.Infof("raw: %v", spew.Sdump(raw.Raw))
	}
	cid := oidcconfig.ClientID{}
	gvk := kinds.ClientID()
	if _, _, err := clientcmdlatest.Codec.Decode(raw.Raw, &gvk, &cid); err != nil {
		l.Err = fmt.Errorf("could not decode: %v", err)
		return nil, nil
	}
	cid.TypeMeta.Kind = "ClientID"
	cid.APIVersion = oidcconfig.SchemeGroupVersion.String()
	result.Extensions[NomosOIDCExtensionKey] = &cid
	if glog.V(5) {
		glog.Infof("\n\nresult: %v,\n\ncid: %v", spew.Sdump(result), spew.Sdump(cid))
	}
	return result, &cid
}

func (l *loginFlow) printKubeConfig(authInfo *api.AuthInfo, authInfoName string) {

	k8sconfig := &api.Config{
		AuthInfos: map[string]*api.AuthInfo{authInfoName: authInfo},
	}

	if writeConfig {
		tempKubeConfig := l.TempFile()
		if l.Err != nil {
			return
		}
		l.WriteToFile(k8sconfig, tempKubeConfig.Name())
		if l.Err != nil {
			return
		}
		kubeConfigPath := l.kubeConfigName()
		loadingRules := clientcmd.ClientConfigLoadingRules{
			Precedence: []string{tempKubeConfig.Name(), kubeConfigPath},
		}
		mergedConfig, err := loadingRules.Load()
		if err != nil {
			glog.Errorf("Error merging configs: %v", err)
			return
		}
		l.WriteToFile(mergedConfig, kubeConfigPath)
		if l.Err != nil {
			return
		}
		glog.Infof("Configuration has been written to %s\n", kubeConfigPath)
	} else {
		glog.Info("\n# Add the following to your ~/.kube/config")

		var json []byte
		json, err := runtime.Encode(clientcmdlatest.Codec, k8sconfig)
		if err != nil {
			l.Err = fmt.Errorf("unexpected error: %v", err)
			return
		}
		var output []byte
		output, err = yaml.JSONToYAML(json)
		if err != nil {
			l.Err = fmt.Errorf("unexpected error: %v", err)
			return
		}
		glog.Infof("%v", string(output))
	}
}

func (l *loginFlow) getAuthorizationCode() string {
	// Start up local webserver at redirect URI to intercept authorization code.
	var code string
	m := http.NewServeMux()
	s := http.Server{Addr: fmt.Sprintf("localhost:%d", port), Handler: m}
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code = r.FormValue("code")
		_, err := w.Write([]byte("done"))
		if err != nil {
			glog.Errorf("Error writing response: %v", err)
			return
		}
		go func() {
			if err = s.Shutdown(context.Background()); err != nil {
				glog.Errorf("Error shutting down server: %v", err)
				return
			}
		}()
	})

	err := s.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		l.Err = fmt.Errorf("ListenAndServe: %s", err)
		return ""
	}

	return code
}

// Run invokes the browser-based OIDC login flow.  Returns the authinfo obtained for
// the user.
func (l *loginFlow) Run(cfg oidc.Config, issuerURL string) string {
	if l.Err != nil {
		return ""
	}

	client := &http.Client{
		Timeout: time.Second * 30,
	}

	glog.V(2).Infof("json: %+v", client)

	parentContext := context.Background()
	clientContext := oidcclient.ClientContext(parentContext, client)
	provider, err := oidcclient.NewProvider(clientContext, issuerURL)
	if err != nil {
		l.Err = fmt.Errorf("Error doing discovery: %s", err)
		return ""
	}

	oauth2Config := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  fmt.Sprintf("http://localhost:%d", port),

		// Discovery returns the OAuth2 endpoints.
		Endpoint: provider.Endpoint(),

		// "openid" is a required scope for OpenID Connect flows.
		Scopes: strings.Split(scope, "+"),
	}

	url := oauth2Config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	if openBrowser {
		helper.LaunchBrowser(true, url)
	} else {
		glog.Infof("Visit the URL for the auth dialog: %v", url)
	}

	code := l.getAuthorizationCode()
	if l.Err != nil {
		return ""
	}

	// Exchange code for tokens
	tok, err := oauth2Config.Exchange(clientContext, code)
	if err != nil {
		l.Err = fmt.Errorf("Error getting tokens: %s", err)
		return ""
	}

	authInfo := helper.GenerateAuthInfo(
		oauth2Config.ClientID, oauth2Config.ClientSecret, tok.Extra("id_token").(string), tok.RefreshToken)

	// TODO: Generate this name from the token.
	authInfoName := "oidc-user"

	l.printKubeConfig(authInfo, authInfoName)
	return authInfoName
}

func (l *loginFlow) mergeWithKubeConfig(file string) *api.Config {
	if l.Err != nil {
		return nil
	}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.Precedence = append([]string{file}, loadingRules.Precedence...)
	mergedConfig, err := loadingRules.Load()
	if err != nil {
		glog.Errorf("while merging configs: %v", err)
		os.Exit(10)
	}
	return mergedConfig
}

func (l *loginFlow) TempFile() *os.File {
	if l.Err != nil {
		return nil
	}
	f, err := ioutil.TempFile("", "")
	if err != nil {
		l.Err = fmt.Errorf("%s", err)
		return nil
	}
	return f
}

func (l *loginFlow) kubeConfigName() string {
	if l.Err != nil {
		return ""
	}
	usr, err := user.Current()
	if err != nil {
		l.Err = fmt.Errorf("can't find current user: %v", err)
		return ""
	}
	return filepath.Join(usr.HomeDir, ".kube", "config")
}
