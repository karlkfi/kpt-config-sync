package filesystem

import (
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/action"
	"github.com/google/stolos/pkg/client/informers/externalversions"
	listers_v1 "github.com/google/stolos/pkg/client/listers/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/policyimporter/actions"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
)

const resync = time.Minute * 15

type Controller struct {
	policyDir           string
	pollPeriod          time.Duration
	parser              *Parser
	differ              *actions.Differ
	client              meta.Interface
	informerFactory     externalversions.SharedInformerFactory
	policyNodeLister    listers_v1.PolicyNodeLister
	clusterPolicyLister listers_v1.ClusterPolicyLister
	stopChan            chan struct{}
}

func NewController(policyDir string, pollPeriod time.Duration, parser *Parser, client meta.Interface, stopChan chan struct{}) *Controller {
	informerFactory := externalversions.NewSharedInformerFactory(
		client.PolicyHierarchy(), resync)
	differ := actions.NewDiffer(
		actions.NewPolicyNodeActionSpec(
			client.PolicyHierarchy().StolosV1(),
			informerFactory.Stolos().V1().PolicyNodes().Lister()),
		actions.NewClusterPolicyActionSpec(
			client.PolicyHierarchy().StolosV1(),
			informerFactory.Stolos().V1().ClusterPolicies().Lister()))

	return &Controller{
		policyDir:           policyDir,
		pollPeriod:          pollPeriod,
		parser:              parser,
		differ:              differ,
		client:              client,
		informerFactory:     informerFactory,
		policyNodeLister:    informerFactory.Stolos().V1().PolicyNodes().Lister(),
		clusterPolicyLister: informerFactory.Stolos().V1().ClusterPolicies().Lister(),
		stopChan:            stopChan,
	}
}

// Run runs the controller blocking until an error occurs or stopChan is closed.
func (c *Controller) Run() error {
	// Start informers
	c.informerFactory.Start(c.stopChan)
	glog.Infof("Waiting for cache to sync")
	synced := c.informerFactory.WaitForCacheSync(c.stopChan)
	for syncType, ok := range synced {
		if !ok {
			elemType := syncType.Elem()
			return errors.Errorf("Failed to sync %s:%s", elemType.PkgPath(), elemType.Name())
		}
	}
	glog.Infof("Caches synced")

	return c.pollDir()
}

func (c *Controller) pollDir() error {
	glog.Infof("Polling policy dir: %s", c.policyDir)

	// Get the current list of policy objects from API server.
	currentPolicies, err := c.getCurentPolicies()
	if err != nil {
		return err
	}

	currentDir := ""
	ticker := time.NewTicker(c.pollPeriod)

	for {
		select {
		case <-ticker.C:
			// Detect whether symlink has changed.
			newDir, err := filepath.EvalSymlinks(c.policyDir)
			if err != nil {
				return errors.Wrapf(err, "failed to resolve policy dir")
			}
			if currentDir == newDir {
				// No new commits, nothing to do.
				continue
			}
			glog.Infof("Resolved policy dir: %s", newDir)

			// Parse filesystem tree into in-memory PolicyNode and ClusterPolicy objects.
			desiredPolicies, err := c.parser.Parse(newDir)
			if err != nil {
				glog.Warningf("Failed to parse: %v", err)
				continue
			}

			// Calculate the sequence of actions needed to transition from current to desired state.
			actions := c.differ.Diff(*currentPolicies, *desiredPolicies)
			if err := applyActions(actions); err != nil {
				glog.Warningf("Failed to apply actions: %v", err)
				continue
			}

			currentDir = newDir
			currentPolicies = desiredPolicies

		case <-c.stopChan:
			glog.Info("Stop polling")
			return nil
		}
	}
}

func (c *Controller) getCurentPolicies() (*v1.AllPolicies, error) {
	policies := v1.AllPolicies{
		PolicyNodes: make(map[string]v1.PolicyNode),
	}

	pn, err := c.policyNodeLister.List(labels.Everything())
	if err != nil {
		return nil, errors.Wrap(err, "failed to list PolicyNodes")
	}
	for _, n := range pn {
		policies.PolicyNodes[n.Name] = *n.DeepCopy()
	}

	cp, err := c.clusterPolicyLister.List(labels.Everything())
	if err != nil {
		return nil, errors.Wrap(err, "failed to list ClusterPolicies")
	}

	if len(cp) > 1 {
		return nil, errors.New("found more than one ClusterPolicy object")
	}
	if len(cp) == 1 {
		policies.ClusterPolicy = cp[0].DeepCopy()
	}

	return &policies, nil
}

func applyActions(actions []action.Interface) error {
	for _, a := range actions {
		if err := a.Execute(); err != nil {
			return err
		}
	}
	return nil
}
