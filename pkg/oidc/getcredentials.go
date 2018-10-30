package oidc

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/oidc/config"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
)

// NomosOIDCExtensionKey is the key under which Nomos OIDC extensions are
// written in the Kubernetes configuration file.
const NomosOIDCExtensionKey = "oidc.nomos.dev"

var getCredentialsCmd = &cobra.Command{
	Use:     "get-credentials",
	Short:   "Gets the OIDC credentials for a cluster",
	Run:     getCreds,
	PreRunE: getCredsCheck,
}

func getCredsCheck(cmd *cobra.Command, _ []string) error {
	if cluster == "" {
		return fmt.Errorf("value for flag --cluster=... is required")
	}
	if clientID == "" {
		return fmt.Errorf("value for flag --client-file=... is required")
	}
	return nil
}

var (
	cluster  string
	clientID string
)

func init() {
	getCredentialsCmd.Flags().StringVar(&cluster, "cluster",
		os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CLUSTER"),
		"the name of the cluster to configure")
	getCredentialsCmd.Flags().StringVar(&clientID, "client-file",
		os.Getenv("KUBECTL_PLUGINS_LOCAL_FLAG_CLIENT_FILE"),
		"the JSON-formatted credentials file for the OIDC client ID for the cluster")
	rootCmd.AddCommand(getCredentialsCmd)
}

// credsFlow encapsulates all the functions in the flow for getting cluster
// credentials.
type credsFlow struct {
	err error
}

func getCreds(cmd *cobra.Command, _ []string) {
	var f credsFlow

	c := f.Open(clientID)
	creds := f.DecodeJSON(c)
	config := f.LoadKubeConfig(kubeConfig)
	clusterRec := f.Cluster(config, cluster)

	clusterConfig := f.EmbedClientID(cluster, clusterRec, creds)
	output := f.ToYAML(clusterConfig)

	if f.err != nil {
		glog.Errorf("with file %q: %v", clientID, f.err)
		os.Exit(1)
	}

	// If we survived all the conversion traps, we're done and we print out.
	fmt.Println(output)
}

func (f *credsFlow) Open(clientID string) io.Reader {
	if f.err != nil {
		return nil
	}
	c, err := os.Open(clientID)
	if err != nil {
		f.err = fmt.Errorf("could not open client-file: %q: %v", clientID, err)
		return nil
	}
	return c
}

// DecodeJSON decodes a JSON-formatted reader into a config.ClientID.
func (f *credsFlow) DecodeJSON(r io.Reader) config.ClientID {
	var cid config.ClientID
	if f.err != nil {
		return cid
	}
	jsonDec := json.NewDecoder(r)
	if err := jsonDec.Decode(&cid); err != nil {
		f.err = fmt.Errorf("while reading JSON from: %v", err)
		return cid
	}
	cid.TypeMeta.Kind = "ClientID"
	cid.APIVersion = config.SchemeGroupVersion.String()
	glog.V(5).Infof("ClientID: %+v", cid)
	return cid
}

// LoadKubeConfig loads the kubernetes config file from kubeConfig path, or if
// empty, from the default path.
func (f *credsFlow) LoadKubeConfig(kubeConfig string) *api.Config {
	if f.err != nil {
		return nil
	}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	glog.V(6).Infof("loadingRules: %+v", loadingRules)
	if kubeConfig != "" {
		loadingRules.Precedence = append([]string{kubeConfig}, loadingRules.Precedence...)
	}

	config, err := loadingRules.Load()
	if err != nil {
		f.err = fmt.Errorf("could not load config: %v", err)
		return nil
	}
	glog.V(6).Infof("config: %+v", config)
	return config
}

// Cluster returns the cluster record with the given name from the given
// configuration.
func (f *credsFlow) Cluster(config *api.Config, cluster string) *api.Cluster {
	if f.err != nil {
		return nil
	}
	clusterRec, ok := config.Clusters[cluster]
	if !ok {
		f.err = fmt.Errorf("no cluster record: %q", cluster)
		return nil
	}
	glog.V(5).Infof("cluster record: %+v", clusterRec)
	return clusterRec
}

func (f *credsFlow) EmbedClientID(cluster string, clusterRec *api.Cluster, clientID config.ClientID) *api.Config {
	if f.err != nil {
		return nil
	}
	clusterConfig := api.NewConfig()
	clusterConfig.Clusters[fmt.Sprintf("oidc:%v", cluster)] = clusterRec
	clusterConfig.Extensions[NomosOIDCExtensionKey] = &clientID
	return clusterConfig
}

// ToYAML converts the given cluster configuration to a YAML string representation.
func (f *credsFlow) ToYAML(clusterConfig *api.Config) string {
	if f.err != nil {
		return ""
	}
	json, err := runtime.Encode(clientcmdlatest.Codec, clusterConfig)
	if err != nil {
		f.err = fmt.Errorf("while encoding: %v", err)
		return ""
	}
	output, err := yaml.JSONToYAML(json)
	if err != nil {
		f.err = fmt.Errorf("while encoding: %+v", err)
		return ""
	}
	return string(output)
}
