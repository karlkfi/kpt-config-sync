package nomostest

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testGitNamespace = "config-management-system-test"
const testGitServer = "test-git-server"
const testGitServerImage = "gcr.io/nomos-public/git-server:v1.0.0"

var testGitServerSelector = map[string]string{"app": testGitServer}

// installGitServer installs the git-server Pod, and returns a callback that
// waits for the Pod to become available.
//
// The git-server almost always comes up before 40 seconds, but we give it a
// full minute in the callback to be safe.
func installGitServer(t *testing.T, nt *NT) func() error {
	t.Helper()

	objs := gitServer()

	for _, o := range objs {
		err := nt.Create(o)
		if err != nil {
			t.Fatalf("installing %v %s", o.GroupVersionKind(),
				client.ObjectKey{Name: o.GetName(), Namespace: o.GetNamespace()})
		}
	}

	return func() error {
		return Retry(60*time.Second, func() error {
			return nt.Validate(testGitServer, testGitNamespace,
				&appsv1.Deployment{}, isAvailableDeployment)
		})
	}
}

// isAvailableDeployment ensures all of the passed Deployment's replicas are
// available.
func isAvailableDeployment(o core.Object) error {
	d, ok := o.(*appsv1.Deployment)
	if !ok {
		return WrongTypeErr(o, d)
	}

	// The desired number of replicas defaults to 1 if unspecified.
	var want int32 = 1
	if d.Spec.Replicas != nil {
		want = *d.Spec.Replicas
	}

	available := d.Status.AvailableReplicas
	if available != want {
		// Display the full state of the malfunctioning Deployment to aid in debugging.
		jsn, err := json.MarshalIndent(d, "", "  ")
		if err != nil {
			return err
		}
		return fmt.Errorf("%w: got %d available replicas, want %d\n\n%s",
			ErrFailedPredicate, available, want, string(jsn))
	}
	return nil
}

func gitServer() []core.Object {
	return []core.Object{
		gitNamespace(),
		gitService(),
		gitDeployment(),
	}
}

func gitNamespace() *corev1.Namespace {
	return fake.NamespaceObject(testGitNamespace)
}

func gitService() *corev1.Service {
	service := fake.ServiceObject(
		core.Name(testGitServer),
		core.Namespace(testGitNamespace),
	)
	service.Spec.Selector = testGitServerSelector
	service.Spec.Ports = []corev1.ServicePort{{Name: "ssh", Port: 22}}
	return service
}

func gitDeployment() *appsv1.Deployment {
	deployment := fake.DeploymentObject(core.Name(testGitServer),
		core.Namespace(testGitNamespace),
		core.Labels(testGitServerSelector),
	)
	deployment.Spec = appsv1.DeploymentSpec{
		MinReadySeconds: 2,
		Strategy:        appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
		Selector:        &v1.LabelSelector{MatchLabels: testGitServerSelector},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: v1.ObjectMeta{
				Labels: testGitServerSelector,
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{Name: "repos", VolumeSource: corev1.VolumeSource{EmptyDir: nil}},
				},
				Containers: []corev1.Container{
					{
						Name:  testGitServer,
						Image: testGitServerImage,
						Ports: []corev1.ContainerPort{{ContainerPort: 22}},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "repos", MountPath: "/git-server/repos/sot.git"},
						},
					},
				},
				ImagePullSecrets: []corev1.LocalObjectReference{},
			},
		},
	}
	return deployment
}
