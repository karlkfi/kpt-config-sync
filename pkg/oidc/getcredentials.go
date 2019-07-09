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
	Short:   "Gets the OIDC credentials for a cluster.",
	Run:     getCreds,
	PreRunE: getCredsCheck,
}

func getCredsCheck(_ *cobra.Command, _ []string) error {
	if cluster == "" {
		return fmt.Errorf("value for flag --cluster=... is required")
	}
	if clientFile == "" {
		if clientID == "" || clientSecret == "" {
			return fmt.Errorf("must set --client-file or --client-id and --client-secret")
		}
	}
	return nil
}

var (
	cluster      string
	clientFile   string
	clientID     string
	clientSecret string
)

func init() {
	getCredentialsCmd.Flags().StringVar(&cluster, "cluster",
		os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CLUSTER"),
		"the name of the cluster to configure")
	getCredentialsCmd.Flags().StringVar(&clientFile, "client-file",
		os.Getenv("KUBECTL_PLUGINS_LOCAL_FLAG_CLIENT_FILE"),
		"the JSON-formatted credentials file for the OIDC client ID for the cluster")
	getCredentialsCmd.Flags().StringVar(&clientID, "client-id",
		os.Getenv("KUBECTL_PLUGINS_LOCAL_FLAG_CLIENT_ID"),
		"the OIDC client ID for the cluster")
	getCredentialsCmd.Flags().StringVar(&clientSecret, "client-secret",
		os.Getenv("KUBECTL_PLUGINS_LOCAL_FLAG_CLIENT_SECRET"),
		"the OIDC client secret for the cluster")
	rootCmd.AddCommand(getCredentialsCmd)
}

// credsFlow encapsulates all the functions in the flow for getting cluster
// credentials.
type credsFlow struct {
	err error
}

func getCreds(_ *cobra.Command, _ []string) {
	var f credsFlow

	cid := f.getClientID()
	cfg := f.LoadKubeConfig(kubeConfig)
	clusterRec := f.Cluster(cfg, cluster)

	clusterConfig := f.embedClientID(cluster, clusterRec, cid)
	output := f.ToYAML(clusterConfig)

	if f.err != nil {
		glog.Errorf("with file %q: %v", clientFile, f.err)
		os.Exit(1)
	}

	// If we survived all the conversion traps, we're done and we print out.
	fmt.Println(output)
}

func (f *credsFlow) getClientID() config.ClientID {
	var cid config.ClientID

	if clientFile != "" {
		c := f.open(clientFile)
		cid = f.DecodeJSON(c)
	} else {
		cid.TypeMeta.Kind = "ClientID"
		cid.APIVersion = config.SchemeGroupVersion.String()
		cid.Installed = &config.InstalledClientSpec{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		}
	}

	return cid
}

func (f *credsFlow) open(clientID string) io.Reader {
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

	cfg, err := loadingRules.Load()
	if err != nil {
		f.err = fmt.Errorf("could not load config: %v", err)
		return nil
	}
	glog.V(6).Infof("config: %+v", cfg)
	return cfg
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

func (f *credsFlow) embedClientID(cluster string, clusterRec *api.Cluster, clientID config.ClientID) *api.Config {
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
	jsonContent, err := runtime.Encode(clientcmdlatest.Codec, clusterConfig)
	if err != nil {
		f.err = fmt.Errorf("while encoding: %v", err)
		return ""
	}
	output, err := yaml.JSONToYAML(jsonContent)
	if err != nil {
		f.err = fmt.Errorf("while encoding: %+v", err)
		return ""
	}
	return string(output)
}
