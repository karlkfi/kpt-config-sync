package vet

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// APIResourcesPath is the path from policyDir to the cached API Resources.
var APIResourcesPath = cmpath.RelativeSlash("api-resources.txt")

// APIResourcesCommand is the command users should run in policyDir to cache
// the API Resources they want to be available to nomos vet.
var APIResourcesCommand = fmt.Sprintf("kubectl api-resources > %s", APIResourcesPath.SlashPath())

// AddCachedAPIResources adds the API Resources from the output of kubectl api-resources
// and adds them to the passed Scoper.
//
// scoper is the Scoper to add resources to.
// file is the file to read API Resources from.
func AddCachedAPIResources(file cmpath.Absolute) discovery.AddResourcesFunc {
	return func(scoper *discovery.Scoper) status.MultiError {
		data, err := ioutil.ReadFile(file.OSPath())
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return UnableToReadAPIResources(file, err)
		}

		errs := addLines(scoper, file, string(data))
		return errs
	}
}

// addLines parses lines of the output of `kubectl api-resources` and assigns
// the provided GKs their assigned scopes in the passed Scoper.
//
// data must begin with the header line, looking similar to:
//   NAME   SHORTNAMES   APIGROUP   NAMESPACED   KIND
func addLines(scoper *discovery.Scoper, path cmpath.Absolute, data string) status.MultiError {
	lines := strings.Split(data, "\n")

	apiGroup := strings.Index(lines[0], "APIGROUP")
	if apiGroup == -1 {
		return MissingAPIGroup(path)
	}
	lines = lines[1:]
	for _, line := range lines {
		if len(line) == 0 {
			// Ignore empty lines.
			continue
		}
		err := addLine(scoper, path, line[apiGroup:])
		if err != nil {
			return err
		}
	}

	return nil
}

// addLine parses a line of `kubectl api-resources` which only has the APIGROUP,
// NAMESPACED, and KIND columns. The resulting GK are assigned the designated
// scope in the passed Scoper.
//
// Returns an error if the NAMESPACED column has a value other than "true" or
// "false".
func addLine(scoper *discovery.Scoper, path cmpath.Absolute, line string) status.MultiError {
	fields := strings.Fields(line)
	kind := fields[len(fields)-1]
	namespaced := fields[len(fields)-2]
	group := ""
	if len(fields) > 2 {
		group = fields[0]
	}

	switch namespaced {
	case "true":
		scoper.SetGroupKindScope(schema.GroupKind{Group: group, Kind: kind}, discovery.NamespaceScope)
		return nil
	case "false":
		scoper.SetGroupKindScope(schema.GroupKind{Group: group, Kind: kind}, discovery.ClusterScope)
		return nil
	default:
		return InvalidScopeValue(path, line, namespaced)
	}
}

// InvalidAPIResourcesCode represents that we were unable to parse the
// api-resources.txt in a repo for some reason.
const InvalidAPIResourcesCode = "1064"

var invalidAPIResourcesBuilder = status.NewErrorBuilder(InvalidAPIResourcesCode)

// UnableToReadAPIResources represents that api-resources.txt exists, but we were
// unable to read it from the disk for some reason.
func UnableToReadAPIResources(path cmpath.Absolute, err error) status.Error {
	return invalidAPIResourcesBuilder.Wrap(err).Sprint("unable to read cached API resources").BuildWithPaths(path)
}

// InvalidScopeValue means that a line had an unexpected scope for its type.
func InvalidScopeValue(path cmpath.Absolute, line, value string) status.Error {
	return invalidAPIResourcesBuilder.Sprintf("invalid NAMESPACED column value %q in line:\n%s\n\nRe-run %q in the root policy directory", value, line, APIResourcesCommand).BuildWithPaths(path)
}

// MissingAPIGroup means that the api-resources.txt is either missing the header
// row, or was generated with an option that omitted the APIGROUP column.
func MissingAPIGroup(path cmpath.Absolute) status.Error {
	return invalidAPIResourcesBuilder.Sprintf("unable to find APIGROUP column. Re-run %q in the root policy directory", APIResourcesCommand).BuildWithPaths(path)
}
