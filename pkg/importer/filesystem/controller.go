package filesystem

import (
	"path"
	"time"

	"github.com/golang/glog"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "git-importer"
	pollFilesystem = "poll-filesystem"
)

// AddController adds the git-importer controller to the manager.
func AddController(mgr manager.Manager, gitDir, policyDirRelative string, pollPeriod time.Duration) error {
	client := syncerclient.New(mgr.GetClient(), metrics.APICallDuration)

	policyDir := path.Join(gitDir, policyDirRelative)
	glog.Infof("Policy dir: %s", policyDir)

	parser := NewParser(
		&genericclioptions.ConfigFlags{}, ParserOpt{Validate: true, Extension: &NomosVisitorProvider{}})
	if err := parser.ValidateInstallation(); err != nil {
		return err
	}

	r, err := NewReconciler(policyDir, parser, client, mgr.GetCache())
	if err != nil {
		return errors.Wrap(err, "failure creating reconciler")
	}
	c, err := controller.New(controllerName, mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return errors.Wrap(err, "failure creating controller")
	}

	return watchFileSystem(c, pollPeriod)
}

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
