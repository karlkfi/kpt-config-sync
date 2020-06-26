package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// RepoSyncReconciler reconciles a RepoSync object
type RepoSyncReconciler struct {
	client client.Client
	log    logr.Logger
	scheme *runtime.Scheme
}

// NewRepoSyncReconciler returns a new RepoSyncReconciler.
func NewRepoSyncReconciler(c client.Client, l logr.Logger, s *runtime.Scheme) *RepoSyncReconciler {
	return &RepoSyncReconciler{
		client: c,
		log:    l,
		scheme: s,
	}
}

// +kubebuilder:rbac:groups=configmanagement.gke.io,resources=reposyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=configmanagement.gke.io,resources=reposyncs/status,verbs=get;update;patch

// Reconcile the RepoSync resource.
func (r *RepoSyncReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// TODO b/160179150 Pass context from the binary where the controllers are registered.
	ctx := context.TODO()
	log := r.log.WithValues("reposync", req.NamespacedName)

	var repoSync v1.RepoSync
	if err := r.client.Get(ctx, req.NamespacedName, &repoSync); err != nil {
		log.Info("unable to fetch RepoSync", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var op controllerutil.OperationResult
	var err error

	if op, err = r.upsertConfigMap(ctx, req, repoSync); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "ConfigMap reconcile failed")
	}
	log.Info("ConfigMap successfully reconciled", executedOperation, op)

	if op, err = r.upsertDeployment(ctx, req, repoSync); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Deployment reconcile failed")
	}
	log.Info("Deployment successfully reconciled", executedOperation, op)

	return ctrl.Result{}, nil
}

// SetupWithManager registers RepoSync controller with reconciler-manager.
func (r *RepoSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.RepoSync{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *RepoSyncReconciler) upsertConfigMap(ctx context.Context, req ctrl.Request, repoSync v1.RepoSync) (controllerutil.OperationResult, error) {
	// CreateOrUpdate() takes a callback, “mutate”, which is where all changes to
	// the object must be performed.
	// The name and namespace  must be filled in prior to calling CreateOrUpdate()
	//
	// Under the hood, CreateOrUpdate() first calls Get() on the object. If the
	// object does not exist, Create() will be called. If it does exist, Update()
	// will be called. Just before calling either Create() or Update(), the mutate
	// callback will be called.
	var childCM corev1.ConfigMap
	childCM.Name = namespaceReconcilerPrefix + req.Namespace
	childCM.Namespace = v1.NSConfigManagementSystem
	op, err := ctrl.CreateOrUpdate(ctx, r.client, &childCM, func() error {
		mutateConfigMap(repoSync, &childCM)
		return nil
	})
	if err != nil {
		return "", err
	}
	return op, nil
}

func mutateConfigMap(rs v1.RepoSync, cm *corev1.ConfigMap) {
	// OwnerReferences, so that when the RepoSync CustomResource is deleted,
	// the corresponding ConfigMap is also deleted.
	cm.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion:         v1.SchemeGroupVersion.String(),
			Kind:               rs.GroupVersionKind().Kind,
			Name:               rs.Name,
			Controller:         pointer.BoolPtr(true),
			BlockOwnerDeletion: pointer.BoolPtr(true),
			UID:                rs.UID,
		},
	}
	cm.Data = map[string]string{
		"GIT_KNOWN_HOSTS": "false",
		"GIT_SYNC_BRANCH": rs.Spec.Revision,
		"GIT_SYNC_REPO":   rs.Spec.Repo,
		"GIT_SYNC_REV":    "HEAD",
		"GIT_SYNC_WAIT":   "15",
	}
}

func (r *RepoSyncReconciler) upsertDeployment(ctx context.Context, req ctrl.Request, repoSync v1.RepoSync) (controllerutil.OperationResult, error) {
	var childDep appsv1.Deployment
	childDep.Name = namespaceReconcilerPrefix + req.Namespace
	childDep.Namespace = v1.NSConfigManagementSystem
	op, err := ctrl.CreateOrUpdate(ctx, r.client, &childDep, func() error {
		mutateDeployment(repoSync, &childDep)
		return nil
	})
	if err != nil {
		return "", err
	}
	return op, nil
}

func mutateDeployment(rs v1.RepoSync, de *appsv1.Deployment) {
	// OwnerReferences, so that when the RepoSync CustomResource is deleted,
	// the corresponding Deployment is also deleted.
	de.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion:         v1.SchemeGroupVersion.String(),
			Kind:               rs.GroupVersionKind().Kind,
			Name:               rs.Name,
			Controller:         pointer.BoolPtr(true),
			BlockOwnerDeletion: pointer.BoolPtr(true),
			UID:                rs.UID,
		},
	}

	// TODO add necessary annotations.

	createOrUpdateLabels(de)

	de.Spec.MinReadySeconds = 10
	de.Spec.Replicas = pointer.Int32Ptr(1)
	de.Spec.Strategy = appsv1.DeploymentStrategy{
		Type: "Recreate",
	}
	var ls metav1.LabelSelector
	de.Spec.Selector = metav1.AddLabelToSelector(&ls, "app", "git-syncer")

	de.Spec.Template.Labels = map[string]string{
		"app": "git-syncer",
	}

	templateSpec := &de.Spec.Template.Spec
	templateSpec.ServiceAccountName = "git-sync"
	templateSpec.SecurityContext = &corev1.PodSecurityContext{
		FSGroup: pointer.Int64Ptr(65533),
	}
	templateSpec.Volumes = []corev1.Volume{
		{
			Name: "repo",
		},
		{
			Name: "git-creds",
			// TODO Add secret
		},
	}

	if len(templateSpec.Containers) == 0 {
		templateSpec.Containers = make([]corev1.Container, 1)
	}

	// TODO Update the containers once git-importer and syncer are merged.
	container := &templateSpec.Containers[0]

	container.Name = "git-sync"
	container.Image = "gcr.io/config-management-release/git-sync:v3.1.6-gke.0__linux_amd64"
	container.Args = []string{
		"--root=/repo",
		"--dest=rev",
		"--max-sync-failures=30",
		"--v=5",
	}
	container.VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "repo",
			MountPath: "/repo",
		},
		{
			Name:      "git-creds",
			ReadOnly:  true,
			MountPath: "/etc/git-secret",
		},
	}
	container.SecurityContext = &corev1.SecurityContext{
		RunAsUser: pointer.Int64Ptr(65533),
	}
	container.EnvFrom = []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: namespaceReconcilerPrefix + rs.Namespace,
				},
			},
		},
	}
}

func createOrUpdateLabels(de *appsv1.Deployment) {
	labels := de.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["app"] = "git-syncer"
	labels["configmanagement.gke.io/system"] = "true"
	de.SetLabels(labels)
}
