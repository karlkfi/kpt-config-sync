package version

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/version"
)

const configManagementVersionName = "configManagementVersion"

func init() {
	flags.AddContexts(Cmd)
	Cmd.Flags().DurationVar(&clientTimeout, "timeout", 3*time.Second, "Timeout for connecting to each cluster")
}

// GetVersionReadCloser returns a ReadCloser with the output produced by running the "nomos version" command as a string
func GetVersionReadCloser(contexts []string) (io.ReadCloser, error) {
	r, w, _ := os.Pipe()
	writer := util.NewWriter(w)
	allCfgs, err := allKubectlConfigs()
	if err != nil {
		return nil, err
	}

	versionInternal(allCfgs, writer, contexts)
	err = w.Close()
	if err != nil {
		return nil, errors.Wrap(err, "failed to close version file writer with error: %v")
	}

	return ioutil.NopCloser(r), nil
}

var (
	// clientTimeout is a flag value to specify how long to wait before timeout of client connection.
	clientTimeout time.Duration

	// clientVersion is a function that obtains the local client version.
	clientVersion = func() string {
		return version.VERSION
	}

	// Cmd is the Cobra object representing the nomos version command.
	Cmd = &cobra.Command{
		Use:   "version",
		Short: "Prints the version of ACM for each cluster as well this CLI",
		Long: `Prints the version of Configuration Management installed on each cluster and the version
of the "nomos" client binary for debugging purposes.`,
		Example: `  nomos version`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Don't show usage on error, as argument validation passed.
			cmd.SilenceUsage = true

			allCfgs, err := allKubectlConfigs()
			versionInternal(allCfgs, os.Stdout, flags.Contexts)

			if err != nil {
				return errors.Wrap(err, "unable to parse kubectl config")
			}
			return nil
		},
	}
)

// allKubectlConfigs gets all kubectl configs, with error handling
func allKubectlConfigs() (map[string]*rest.Config, error) {
	allCfgs, err := restconfig.AllKubectlConfigs(clientTimeout)
	if err != nil {
		// Unwrap the "no such file or directory" error for better readability
		if unWrapped := errors.Cause(err); os.IsNotExist(unWrapped) {
			err = unWrapped
		}

		// nolint:errcheck
		fmt.Printf("failed to create client configs: %v\n", err)
	}

	return allCfgs, err
}

// versionInternal allows stubbing out the config for tests.
func versionInternal(configs map[string]*rest.Config, w io.Writer, contexts []string) {
	// See go/nomos-cli-version-design for the output below.
	if contexts != nil {
		// filter by specified contexts
		configs = filterConfigs(contexts, configs)
		if len(configs) == 0 {
			fmt.Print("No clusters match the specified context.\n")
		}
	}

	vs := versions(configs)
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

func lookupVersion(cfg *rest.Config) (string, error) {
	cmClient, err := util.NewConfigManagementClient(cfg)
	if err != nil {
		return util.ErrorMsg, err
	}
	// {
	//   ...
	//   "status": {
	//     ...
	//     "configManagementVersion": "some_string",
	//     ...
	//   }
	// }
	cmVersion, err := cmClient.NestedString(context.Background(), "status", configManagementVersionName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return util.NotInstalledMsg, nil
		}
		return util.ErrorMsg, err
	}
	if cmVersion == "" {
		cmVersion = util.UnknownMsg
	}
	return cmVersion, nil
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
			ve.version, ve.err = lookupVersion(c)
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
	// current denotes the current context. the value is a '*' if this is the current context,
	// or any empty string otherwise
	current string
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
	currentContext, err := restconfig.CurrentContextName()
	if err != nil {
		fmt.Printf("Failed to get current context name with err: %v\n", errors.Cause(err))
	}

	var es []entry
	for n, v := range vs {
		curr := ""
		if n == currentContext && err == nil {
			curr = "*"
		}
		es = append(es, entry{current: curr, name: n, component: util.ConfigManagementName, vErr: v})
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
	format := "%s\t%s\t%s\t%s\n"
	w := util.NewWriter(out)
	defer func() {
		if err := w.Flush(); err != nil {
			// nolint:errcheck
			fmt.Fprintf(os.Stderr, "error on Flush(): %v", err)
		}
	}()
	// nolint:errcheck
	fmt.Fprintf(w, format, "CURRENT", "NAME", "COMPONENT", "VERSION")
	for _, e := range es {
		if e.err != nil {
			// nolint:errcheck
			fmt.Fprintf(w, format, e.current, e.name, e.component, fmt.Sprintf("<error: %v>", e.err))
			continue
		}
		// nolint:errcheck
		fmt.Fprintf(w, format, e.current, e.name, e.component, e.version)
	}
}
