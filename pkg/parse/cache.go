package parse

import (
	"time"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// cacheForCommit tracks the progress made by the reconciler for a git commit.
//
// The reconciler resets the whole cache when a new git commit is detected.
//
// The reconciler resets the whole cache except for the cached gitState when:
//   * a force-resync happens, or
//   * one of the watchers noticed a management conflict.
type cacheForCommit struct {
	// git tracks the state of the git repo.
	// This field is only set after the reconciler successfully reads all the git files.
	git gitState

	// hasParserResult indicates whether the cache includes the parser result.
	//
	// An alternative is to determining whether the cache includes the parser result by
	// checking whether the `parserResult` field is empty. However, an empty `parserResult`
	// field may also indicate that the git repo is empty.
	hasParserResult bool

	// parserResult contains the parser result.
	// The field is only set when the parser succeeded.
	parserResult []ast.FileObject

	// resourceDeclSetUpdated indicates whether the resource declaration set has been updated.
	resourceDeclSetUpdated bool

	// hasApplierResult indicates whether the cache includes the parser result.
	//
	// An alternative is to determining whether the cache includes the parser result by
	// checking whether the `applierResult` field is empty. However, an empty `applierResult`
	// field may also indicate that the git repo is empty and there is nothing to be applied.
	hasApplierResult bool

	// applierResult contains the applier result.
	// The field is only set when the applier succeeded applied all the declared resources.
	applierResult map[schema.GroupVersionKind]struct{}

	// needToRetry indicates whether a retry is needed.
	needToRetry bool

	// reconciliationWithSameErrs tracks the number of reconciliation attempts failed with the same errors.
	reconciliationWithSameErrs int

	// nextRetryTime tracks when the next retry should happen.
	nextRetryTime time.Time

	// errs tracks all the errors encounted during the reconciliation.
	errs status.MultiError
}

func (c *cacheForCommit) setParserResult(result []ast.FileObject) {
	c.hasParserResult = true
	c.parserResult = result
}

func (c *cacheForCommit) setApplierResult(result map[schema.GroupVersionKind]struct{}) {
	c.hasApplierResult = true
	c.applierResult = result
}

func (c *cacheForCommit) readyToRetry() bool {
	return !time.Now().Before(c.nextRetryTime)
}
