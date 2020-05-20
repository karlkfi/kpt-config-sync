package testoutput

import (
	"strconv"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
)

// DepthLabels labels namespaces with depths to other hierarchy.
func DepthLabels(path string) core.MetaMutator {
	tl := make(map[string]string)
	p := strings.Split(path, "/")
	p = append([]string{v1.DepthLabelRootName}, p...)
	for i, ans := range p {
		l := ans + v1.HierarchyControllerDepthSuffix
		dist := strconv.Itoa(len(p) - i - 1)
		tl[l] = dist
	}
	return core.Labels(tl)
}
