package filesystem

import (
	"os"
	"path"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/syncer/decode"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
)

const (
	controllerName = "git-importer"
	// pollFilesystem is an invalid resource name used to signal that the event
	// triggering the reconcile is a periodic poll of the filesystem. The reason
	// it is an invalid name is we want to prevent treating a namespaceconfig
	// change as a poll filesystem event, if it happens to be named poll-filesystem.
	pollFilesystem = "@poll-filesystem"
)

// AddController adds the git-importer controller to the manager.
func AddController(clusterName string, mgr manager.Manager, gitDir, policyDirRelative string, pollPeriod time.Duration) error {
	client := syncerclient.New(mgr.GetClient(), metrics.APICallDuration)

	rootDir := path.Join(gitDir, policyDirRelative)
	glog.Infof("Policy dir: %s", rootDir)

	var err error
	rootPath, err := cmpath.NewRoot(cmpath.FromOS(rootDir))
	if err != nil {
		return err
	}

	if err = ValidateInstallation(importer.DefaultCLIOptions); err != nil {
		return err
	}

	dc, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrapf(err, "failed to create discoveryclient")
	}

	// If HIERARCHY_DISABLED is invalid, assume disabled.
	hierarchyDisabled, _ := strconv.ParseBool(os.Getenv("HIERARCHY_DISABLED"))
	var cfgParser configParser
	if hierarchyDisabled {
		// Nomos hierarchy is disabled, so use the RawParser.
		cfgParser = NewRawParser(rootPath.Join(cmpath.FromSlash(".")), &FileReader{ClientGetter: importer.DefaultCLIOptions}, importer.DefaultCLIOptions)
	} else {
		cfgParser = NewParser(importer.DefaultCLIOptions, ParserOpt{Extension: &NomosVisitorProvider{}, RootPath: rootPath})
	}

	decoder := decode.NewGenericResourceDecoder(runtime.NewScheme())
	r, err := NewReconciler(clusterName, rootDir, cfgParser, client, dc, mgr.GetCache(), decoder)
	if err != nil {
		return errors.Wrap(err, "failure creating reconciler")
	}
	c, err := controller.New(controllerName, mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return errors.Wrap(err, "failure creating controller")
	}

	// We map all requests generated from from watching Nomos CRs to the same request.
	// The reason we do this is because the logic is the same in the reconcile loop,
	// regardless of which resource changed. Having a constant used for the reconcile.Request
	// avoids doing redundant reconciles.
	mapToConstant := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(nomosResourceRequest),
	}

	// Watch all Nomos CRs that are managed by the importer.
	if err = c.Watch(&source.Kind{Type: &v1.ClusterConfig{}}, mapToConstant); err != nil {
		return errors.Wrapf(err, "could not watch ClusterConfigs in the %q controller", controllerName)
	}
	if err = c.Watch(&source.Kind{Type: &v1.NamespaceConfig{}}, mapToConstant); err != nil {
		return errors.Wrapf(err, "could not watch NamespaceConfigs in the %q controller", controllerName)
	}
	if err = c.Watch(&source.Kind{Type: &v1.Sync{}}, mapToConstant); err != nil {
		return errors.Wrapf(err, "could not watch Syncs in the %q controller", controllerName)
	}

	return watchFileSystem(c, pollPeriod)
}

// watchFileSystem issues a reconcile.Request after every pollPeriod.
func watchFileSystem(c controller.Controller, pollPeriod time.Duration) error {
	pollCh := make(chan event.GenericEvent)
	go func() {
		ticker := time.NewTicker(pollPeriod)
		for range ticker.C {
			pollCh <- event.GenericEvent{Meta: &metav1.ObjectMeta{Name: pollFilesystem}}
		}
	}()

	pollSource := &source.Channel{Source: pollCh}
	if err := c.Watch(pollSource, &handler.EnqueueRequestForObject{}); err != nil {
		return errors.Wrapf(err, "could not watch manager initialization errors in the %q controller", controllerName)
	}

	return nil
}

// nomosResourceRequest maps resources being watched,
// to reconciliation requests for a cluster-scoped resource with name "nomos-resource".
func nomosResourceRequest(_ handler.MapObject) []reconcile.Request {
	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			Name: "nomos-resource",
		},
	}}
}
