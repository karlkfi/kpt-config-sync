package parse

import (
	"github.com/google/nomos/pkg/core"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// cache tracks the progress made by the updater
type cache struct {
	git gitState

	// hasParserResult indicates whether the cache includes the parser result
	//
	// An alternative is to determining whether the cache includes the parser result by
	// checking whether the `parserResult` field is empty. However, an empty `parserResult`
	// field may also indicate that the git repo is empty.
	hasParserResult bool

	// parserResult contains the parser result
	// The field is only set when the parser succeeded.
	parserResult []core.Object

	// resourceDeclSetUpdated indicates whether the resource declaration set has been updated
	resourceDeclSetUpdated bool

	// hasApplierResult indicates whether the cache includes the parser result
	//
	// An alternative is to determining whether the cache includes the parser result by
	// checking whether the `applierResult` field is empty. However, an empty `applierResult`
	// field may also indicate that the git repo is empty and there is nothing to be applied.
	hasApplierResult bool

	// applierResult contains the applier result
	// The field is only set when the applier succeeded applied all the declared resources.
	applierResult map[schema.GroupVersionKind]struct{}

	// sourceStatusUpdated indicates whether the `Status.Source` field of a RepoSync/RootSync has
	// been updated successfully
	sourceStatusUpdated bool

	// syncStatusUpdated indicates whether the `Status.Sync` field of a RepoSync/RootSync has
	// been updated successfully
	syncStatusUpdated bool

	// needToRetry indicates whether a retry is needed
	needToRetry bool
}

func (c *cache) setParserResult(result []core.Object) {
	c.hasParserResult = true
	c.parserResult = result
}

func (c *cache) setApplierResult(result map[schema.GroupVersionKind]struct{}) {
	c.hasApplierResult = true
	c.applierResult = result
}
