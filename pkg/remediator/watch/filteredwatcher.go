package watch

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/remediator/queue"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/differ"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// Copying strategy from k8s.io/client-go/tools/cache/reflector.go
	// We try to spread the load on apiserver by setting timeouts for
	// watch requests - it is random in [minWatchTimeout, 2*minWatchTimeout].
	minWatchTimeout = 5 * time.Minute
)

// maxWatchRetryFactor is used to determine when the next retry should happen.
// 2^^18 * time.Millisecond = 262,144 ms, which is about 4.36 minutes.
const maxWatchRetryFactor = 18

// Runnable defines the custom watch interface.
type Runnable interface {
	Stop()
	Run(ctx context.Context) status.Error
	ManagementConflict() bool
	SetManagementConflict(v bool)
}

const (
	watchEventBookmarkType    = "Bookmark"
	watchEventErrorType       = "Error"
	watchEventUnsupportedType = "Unsupported"
)

// errorLoggingInterval specifies the minimal time interval two errors related to the same object
// and having the same errorType should be logged.
const errorLoggingInterval = time.Second

// filteredWatcher is wrapper around a watch interface.
// It only keeps the events for objects that are
// - either present in the declared resources,
// - or managed by Config Sync.
type filteredWatcher struct {
	gvk        string
	startWatch startWatchFunc
	resources  *declared.Resources
	queue      *queue.ObjectQueue
	reconciler declared.Scope
	// errorTracker maps an error to the time when the same error happened last time.
	errorTracker map[string]time.Time

	// The following fields are guarded by the mutex.
	mux                sync.Mutex
	base               watch.Interface
	stopped            bool
	managementConflict bool
}

// filteredWatcher implements the Runnable interface.
var _ Runnable = &filteredWatcher{}

// NewFiltered returns a new filtered watch initialized with the given options.
func NewFiltered(_ context.Context, cfg watcherConfig) Runnable {
	return &filteredWatcher{
		gvk:          cfg.gvk.String(),
		startWatch:   cfg.startWatch,
		resources:    cfg.resources,
		queue:        cfg.queue,
		reconciler:   cfg.reconciler,
		base:         watch.NewEmptyWatch(),
		errorTracker: make(map[string]time.Time),
	}
}

// pruneErrors removes the errors happened before errorLoggingInterval from w.errorTracker.
// This is to save the memory usage for tracking errors.
func (w *filteredWatcher) pruneErrors() {
	for errName, lastErrorTime := range w.errorTracker {
		if time.Since(lastErrorTime) >= errorLoggingInterval {
			delete(w.errorTracker, errName)
		}
	}
}

// addError checks whether an error identified by the errorID has been tracked,
// and handles it in one of the following ways:
//   * tracks it if it has not yet been tracked;
//   * updates the time for this error to time.Now() if `errorLoggingInterval` has passed
//     since the same error happened last time;
//   * ignore the error if `errorLoggingInterval` has NOT passed since it happened last time.
//
// addError returns false if the error is ignored, and true if it is not ignored.
func (w *filteredWatcher) addError(errorID string) bool {
	lastErrorTime, ok := w.errorTracker[errorID]
	if !ok || time.Since(lastErrorTime) >= errorLoggingInterval {
		w.errorTracker[errorID] = time.Now()
		return true
	}
	return false
}

func (w *filteredWatcher) ManagementConflict() bool {
	w.mux.Lock()
	defer w.mux.Unlock()
	return w.managementConflict
}

func (w *filteredWatcher) SetManagementConflict(v bool) {
	w.mux.Lock()
	w.managementConflict = v
	w.mux.Unlock()
}

// Stop fully stops the filteredWatcher in a threadsafe manner. This means that
// it stops the underlying base watch and prevents the filteredWatcher from
// restarting it (like it does if the API server disconnects the base watch).
func (w *filteredWatcher) Stop() {
	w.mux.Lock()
	defer w.mux.Unlock()

	w.base.Stop()
	w.stopped = true
}

// This function is borrowed from https://github.com/kubernetes/client-go/blob/master/tools/cache/reflector.go.
func isExpiredError(err error) bool {
	// In Kubernetes 1.17 and earlier, the api server returns both apierrors.StatusReasonExpired and
	// apierrors.StatusReasonGone for HTTP 410 (Gone) status code responses. In 1.18 the kube server is more consistent
	// and always returns apierrors.StatusReasonExpired. For backward compatibility we can only remove the apierrors.IsGone
	// check when we fully drop support for Kubernetes 1.17 servers from reflectors.
	return apierrors.IsResourceExpired(err) || apierrors.IsGone(err)
}

// TODO(b/184078084): Use wait.ExponentialBackoff in the watch retry logic
func waitUntilNextRetry(retries int) {
	if retries > maxWatchRetryFactor {
		retries = maxWatchRetryFactor
	}
	milliseconds := int64(math.Pow(2, float64(retries)))
	duration := time.Duration(milliseconds) * time.Millisecond
	time.Sleep(duration)
}

// Run reads the event from the base watch interface,
// filters the event and pushes the object contained
// in the event to the controller work queue.
func (w *filteredWatcher) Run(ctx context.Context) status.Error {
	glog.Infof("Watch started for %s", w.gvk)
	var resourceVersion string
	var retriesForWatchError int

	for {
		// There are three ways this function can return:
		// 1. false, error -> We were unable to start the watch, so exit Run().
		// 2. false, nil   -> We have been stopped via Stop(), so exit Run().
		// 3. true,  nil   -> We have not been stopped and we started a new watch.
		started, err := w.start(resourceVersion)
		if err != nil {
			return err
		}
		if !started {
			break
		}

		glog.V(2).Infof("(Re)starting watch for %s at resource version %q", w.gvk, resourceVersion)
		for event := range w.base.ResultChan() {
			w.pruneErrors()
			newVersion, err := w.handle(ctx, event)
			if err != nil {
				if isExpiredError(err) {
					glog.V(2).Infof("Watch for %s at resource version %q closed with: %v", w.gvk, resourceVersion, err)
					// `w.handle` may fail because we try to watch an old resource version, setting
					// a watch on an old resource version will always fail.
					// Reset `resourceVersion` to an empty string here so that we can start a new
					// watch at the most recent resource version.
					resourceVersion = ""
				} else if w.addError(watchEventErrorType + errorID(err)) {
					glog.Errorf("Watch for %s at resource version %q ended with: %v", w.gvk, resourceVersion, err)
				}
				retriesForWatchError++
				waitUntilNextRetry(retriesForWatchError)
				// Call `break` to restart the watch.
				break
			}
			retriesForWatchError = 0
			if newVersion != "" {
				resourceVersion = newVersion
			}
		}
		glog.V(2).Infof("Ending watch for %s at resource version %q", w.gvk, resourceVersion)
	}
	glog.Infof("Watch stopped for %s", w.gvk)
	return nil
}

// start initiates a new base watch at the given resource version in a
// threadsafe manner and returns true if the new base watch was created. Returns
// false if the filteredWatcher is already stopped and returns error if the base
// watch could not be started.
func (w *filteredWatcher) start(resourceVersion string) (bool, status.Error) {
	w.mux.Lock()
	defer w.mux.Unlock()

	if w.stopped {
		return false, nil
	}
	w.base.Stop()

	// We want to avoid situations of hanging watchers. Stop any watchers that
	// do not receive any events within the timeout window.
	timeoutSeconds := int64(minWatchTimeout.Seconds() * (rand.Float64() + 1.0))
	options := metav1.ListOptions{
		AllowWatchBookmarks: true,
		ResourceVersion:     resourceVersion,
		TimeoutSeconds:      &timeoutSeconds,
		Watch:               true,
	}

	base, err := w.startWatch(options)
	if err != nil {
		return false, status.APIServerErrorf(err, "failed to start watch for %s", w.gvk)
	}
	w.base = base
	return true, nil
}

func errorID(err error) string {
	errTypeName := reflect.TypeOf(err).String()

	var s string
	switch t := err.(type) {
	case *apierrors.StatusError:
		if t == nil {
			break
		}
		if t.ErrStatus.Details != nil {
			s = t.ErrStatus.Details.Name
		}
		if s == "" {
			s = fmt.Sprintf("%s-%s-%d", t.ErrStatus.Status, t.ErrStatus.Reason, t.ErrStatus.Code)
		}
	}
	return errTypeName + s
}

// handle reads the event from the base watch interface,
// filters the event and pushes the object contained
// in the event to the controller work queue.
//
// handle returns the new resource version, and an error indicating that
// an a watch.Error event type is encounted and the watch should be restarted.
func (w *filteredWatcher) handle(ctx context.Context, event watch.Event) (string, error) {
	var deleted bool
	switch event.Type {
	case watch.Added, watch.Modified:
		deleted = false
	case watch.Deleted:
		deleted = true
	case watch.Bookmark:
		m, err := meta.Accessor(event.Object)
		if err != nil {
			// For watch.Bookmark, only the ResourceVersion field of event.Object is set.
			// Therefore, set the second argument of w.addError to watchEventBookmarkType.
			if w.addError(watchEventBookmarkType) {
				glog.Errorf("Unable to access metadata of Bookmark event: %v", event)
			}
			return "", nil
		}
		return m.GetResourceVersion(), nil
	case watch.Error:
		return "", apierrors.FromObject(event.Object)
	// Keep the default case to catch any new watch event types added in the future.
	default:
		if w.addError(watchEventUnsupportedType) {
			glog.Errorf("Unsupported watch event: %#v", event)
		}
		return "", nil
	}

	// get client.Object from the runtime object.
	object, ok := event.Object.(client.Object)
	if !ok {
		glog.Warningf("Received non client.Object in watch event: %T", object)
		metrics.RecordInternalError(ctx, "remediator")
		return "", nil
	}
	// filter objects.
	if !w.shouldProcess(object) {
		glog.V(4).Infof("Ignoring event for object: %v", object)
		return object.GetResourceVersion(), nil
	}

	if deleted {
		glog.V(2).Infof("Received watch event for deleted object %q", core.IDOf(object))
		object = queue.MarkDeleted(ctx, object)
	} else {
		glog.V(2).Infof("Received watch event for created/updated object %q", core.IDOf(object))
	}

	glog.V(3).Infof("Received object: %v", object)
	w.queue.Add(object)
	return object.GetResourceVersion(), nil
}

// shouldProcess returns true if the given object should be enqueued by the
// watcher for processing.
func (w *filteredWatcher) shouldProcess(object client.Object) bool {
	if !diff.CanManage(w.reconciler, object) {
		glog.Infof("Found management conflict for object: %v", object)
		w.SetManagementConflict(true)
		return false
	}

	id := core.IDOf(object)
	if decl, ok := w.resources.Get(id); ok {
		// If the object is declared, we only process it if it has the same GVK as
		// its declaration. Otherwise we expect to get another event for the same
		// object but with a matching GVK so we can actually compare it to its
		// declaration.
		return object.GetObjectKind().GroupVersionKind() == decl.GroupVersionKind()
	}

	// Even if the object is undeclared, we still want to process it if it is
	// tagged as a managed object.
	if !differ.ManagementEnabled(object) {
		return false
	}

	// Only process non declared, managed resources if we are the manager.
	return diff.IsManager(w.reconciler, object)
}
