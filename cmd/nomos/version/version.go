package version

import (
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/version"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const (
	unknown                     = "<unknown>"
	errMsg                      = "<error>"
	notInstalled                = "<not installed>"
	configManagementName        = "config-management"
	configManagementVersionName = "configManagementVersion"
)

func init() {
	flags.AddContexts(Cmd)
}

var (
	// Cmd is the Cobra object representing the nomos version command.
	Cmd = &cobra.Command{
		Use:   "version",
		Short: "Prints the version of this binary",
		Long: `Prints the version of the "nomos" client binary for debugging purposes.
`,
		Example: `  nomos version`,
		Run: func(_ *cobra.Command, _ []string) {
			versionInternal(
				restconfig.AllKubectlConfigs, os.Stdout,
				flags.Contexts)
		},
	}

	// clientVersion is a function that obtains the local client version.
	clientVersion = func() string {
		return version.VERSION
	}

	// dynamicCLient obtains a client based on the supplied REST config.  Can
	// be overridden in tests.
	dynamicClient = dynamic.NewForConfig
)

// versionInternal allows stubbing out the config for tests.
func versionInternal(
	configs func(time.Duration) (map[string]*rest.Config, error),
	w io.Writer, contexts []string) {

	// See go/nomos-cli-version-design for the output below.
	allCfgs, err := configs(30 * time.Second)
	if err != nil {
		// nolint:errcheck
		fmt.Fprintf(w, "could not get kubernetes config file: %v", err)
		os.Exit(255)
	}
	cfgs := allCfgs

	if contexts != nil {
		// filter by specified contexts
		cfgs = filterConfigs(contexts, allCfgs)
	}

	vs := versions(cfgs)
	es := entries(vs)
	tabulate(es, w)
}

// filterConfigs retains from all only the configs that have been selected
// through flag use. contexts is the list of contexts to print information for.
// If allClusters is true, contexts is ignored and information for all contexts
// is printed.
func filterConfigs(contexts []string, all map[string]*rest.Config) map[string]*rest.Config {
	cfgs := make(map[string]*rest.Config)
	for _, name := range contexts {
		if cfg, ok := all[name]; ok {
			cfgs[name] = cfg
		}
	}
	return cfgs
}

func client(name string, c *rest.Config) (dynamic.ResourceInterface, error) {
	cl, err := dynamicClient(c)
	if err != nil {
		return nil, err
	}
	gvr := schema.GroupVersionResource{
		Group:   "addons.sigs.k8s.io",
		Version: "v1alpha1",
		// The dynamic client needs the plural resource form to be able to
		// construct a correct resource URL.
		Resource: "configmanagements",
	}
	return cl.Resource(gvr).Namespace(""), nil
}

func lookupVersion(name string, cfg *rest.Config) (string, error) {
	i, err := client(name, cfg)
	if err != nil {
		return "", err
	}
	u, err := i.Get(configManagementName, metav1.GetOptions{}, "")
	if err != nil {
		if apierrors.IsNotFound(err) {
			return notInstalled, nil
		}
		return errMsg, err
	}
	c := u.UnstructuredContent()
	// {
	//   ...
	//   "status": {
	//     ...
	//     "configManagementVersion": "some_string",
	//     ...
	//   }
	// }
	s, ok := c["status"]
	if !ok {
		return errMsg, fmt.Errorf("internal error: can not parse status")
	}
	sp, ok := s.(map[string]interface{})
	if !ok {
		return errMsg, fmt.Errorf("internal error: status is not a map")
	}
	d, ok := sp[configManagementVersionName]
	if !ok {
		return unknown, nil
	}
	v, ok := d.(string)
	if !ok {
		return errMsg, fmt.Errorf("internal error: configManagementVersion is not a string")
	}
	if v == "" {
		v = unknown
	}
	return v, nil
}

// vErr is either a version or an error.
type vErr struct {
	version string
	err     error
}

// versions obtains the versions of all configmanagements from the contexts
// supplied in the named configs.
func versions(cfgs map[string]*rest.Config) map[string]vErr {
	if len(cfgs) == 0 {
		return nil
	}
	vs := make(map[string]vErr, len(cfgs))
	var (
		m sync.Mutex // GUARDS vs
		g sync.WaitGroup
	)
	for n, c := range cfgs {
		g.Add(1)
		go func(n string, c *rest.Config) {
			defer g.Done()
			var ve vErr
			ve.version, ve.err = lookupVersion(n, c)
			m.Lock()
			vs[n] = ve
			m.Unlock()
		}(n, c)
	}
	g.Wait()
	return vs
}

// entry is one entry of the output
type entry struct {
	// name is the context's name.
	name string
	// component is the nomos component name.
	component string
	// vErr is either a version or an error.
	vErr
}

// entries produces a stable list of version reports based on the unordered
// versions provided.
func entries(vs map[string]vErr) []entry {
	var es []entry
	for n, v := range vs {
		es = append(es, entry{name: n, component: configManagementName, vErr: v})
	}
	// Also fill in the client version here.
	es = append(es, entry{
		component: "<client>",
		vErr:      vErr{version: clientVersion(), err: nil}})
	sort.SliceStable(es, func(i, j int) bool {
		return es[i].name < es[j].name
	})
	return es
}

// tabulate prints out the findings in the provided entries in a nice tabular
// form.  It's the sixties, go for it!
func tabulate(es []entry, out io.Writer) {
	const format = "%s\t%s\t%s\n"
	w := util.NewWriter(out)
	defer func() {
		if err := w.Flush(); err != nil {
			// nolint:errcheck
			fmt.Fprintf(os.Stderr, "error on Flush(): %v", err)
		}
	}()
	// nolint:errcheck
	fmt.Fprintf(w, format, "NAME", "COMPONENT", "VERSION")
	for _, e := range es {
		if e.err != nil {
			// nolint:errcheck
			fmt.Fprintf(w, format, e.name, e.component, fmt.Sprintf("<error: %v>", e.err))
			continue
		}
		// nolint:errcheck
		fmt.Fprintf(w, format, e.name, e.component, e.version)
	}
}
