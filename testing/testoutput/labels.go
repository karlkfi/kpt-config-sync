package testoutput

import (
	"strconv"
	"strings"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/hnc"
)

// DepthLabels labels namespaces with depths to other hierarchy.
func DepthLabels(path string) core.MetaMutator {
	tl := make(map[string]string)
	p := strings.Split(path, "/")
	p = append([]string{hnc.DepthLabelRootName}, p...)
	for i, ans := range p {
		l := ans + hnc.DepthSuffix
		dist := strconv.Itoa(len(p) - i - 1)
		tl[l] = dist
	}
	return core.Labels(tl)
}
