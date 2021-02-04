package parse

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// cacheForCommit tracks the progress made by the reconciler for a git commit
// The reconciler resets `cacheForCommit` when:
//   * a new git commit is detected, or
//   * a force-resync happens, or
//   * one of the watchers noticed a management conflict.
type cacheForCommit struct {
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

	// needToRetry indicates whether a retry is needed
	needToRetry bool

	// errs tracks all the errors encounted during the reconciliation
	errs status.MultiError
}

func (c *cacheForCommit) setParserResult(result []core.Object) {
	c.hasParserResult = true
	c.parserResult = result
}

func (c *cacheForCommit) setApplierResult(result map[schema.GroupVersionKind]struct{}) {
	c.hasApplierResult = true
	c.applierResult = result
}
