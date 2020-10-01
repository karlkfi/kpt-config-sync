package nomostest

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"

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
const rootRepo = "sot.git"

func testGitServerSelector() map[string]string {
	// Note that maps are copied by reference into objects.
	// If this were just a variable, then concurrent usages by Clients may result
	// in concurrent map writes (and thus flaky test panics).
	return map[string]string{"app": testGitServer}
}

// installGitServer installs the git-server Pod, and returns a callback that
// waits for the Pod to become available.
//
// The git-server almost always comes up before 40 seconds, but we give it a
// full minute in the callback to be safe.
func installGitServer(nt *NT) func() error {
	nt.T.Helper()

	objs := gitServer()

	for _, o := range objs {
		err := nt.Create(o)
		if err != nil {
			nt.T.Fatalf("installing %v %s", o.GroupVersionKind(),
				client.ObjectKey{Name: o.GetName(), Namespace: o.GetNamespace()})
		}
	}

	return func() error {
		// In CI, 2% of the time this takes longer than 60 seconds, so 120 seconds
		// seems like a reasonable amount of time to wait before erroring out.
		took, err := Retry(120*time.Second, func() error {
			return nt.Validate(testGitServer, testGitNamespace,
				&appsv1.Deployment{}, isAvailableDeployment)
		})
		if err != nil {
			return err
		}
		nt.T.Logf("took %v to wait for git-server to come up", took)
		return nil
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
	// Remember that we've already created the git-server's Namespace since the
	// SSH key must exist before we apply the Deployment.
	return []core.Object{
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
	service.Spec.Selector = testGitServerSelector()
	service.Spec.Ports = []corev1.ServicePort{{Name: "ssh", Port: 22}}
	return service
}

func gitDeployment() *appsv1.Deployment {
	deployment := fake.DeploymentObject(core.Name(testGitServer),
		core.Namespace(testGitNamespace),
		core.Labels(testGitServerSelector()),
	)

	deployment.Spec = appsv1.DeploymentSpec{
		MinReadySeconds: 2,
		Strategy:        appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
		Selector:        &v1.LabelSelector{MatchLabels: testGitServerSelector()},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: v1.ObjectMeta{
				Labels: testGitServerSelector(),
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{Name: "keys", VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{SecretName: "ssh-pub"},
					}},
					{Name: "repos", VolumeSource: corev1.VolumeSource{EmptyDir: nil}},
				},
				Containers: []corev1.Container{
					{
						Name:  testGitServer,
						Image: testGitServerImage,
						Ports: []corev1.ContainerPort{{ContainerPort: 22}},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "keys", MountPath: "/git-server/keys"},
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

// portForwardGitServer forwards the git-server deployment to a port.
// Returns the localhost port which forwards to the git-server Pod.
func portForwardGitServer(nt *NT, repos ...string) int {
	nt.T.Helper()

	// This logic is not robust to the git-server pod being killed/restarted,
	// but this is a rare occurrence.
	// Consider if it is worth getting the Pod name again if port forwarding fails.
	podList := &corev1.PodList{}
	err := nt.List(podList, client.InNamespace(testGitNamespace))
	if err != nil {
		nt.T.Fatal(err)
	}
	if nPods := len(podList.Items); nPods != 1 {
		podsJSON, err := json.MarshalIndent(podList, "", "  ")
		if err != nil {
			nt.T.Fatal(err)
		}
		nt.T.Log(string(podsJSON))
		nt.T.Fatalf("got len(podList.Items) = %d, want 1", nPods)
	}

	podName := podList.Items[0].Name

	for _, repo := range repos {
		nt.Kubectl("exec", "-n", testGitNamespace, podName, "--",
			"git", "init", "--bare", "--shared", fmt.Sprintf("/git-server/repos/%s", repo))
		// We set receive.denyNonFastforwards to allow force pushes for legacy test support (bats).  In the future we may
		// need this support for testing GKE clusters since we will likely be re-using the cluster in that case.
		// Alternatively, we could also run "rm -rf /git-server/repos/*" to clear out the state of the git server and
		// re-initialize.
		nt.Kubectl("exec", "-n", testGitNamespace, podName, "--",
			"git", "-C", fmt.Sprintf("/git-server/repos/%s", repo), "config", "receive.denyNonFastforwards", "false")
	}
	return forwardToFreePort(nt.T, nt.kubeconfigPath, podName)
}

// forwardToPort forwards the given Pod in the git-server's Namespace to
// a free port.
//
// Returns the localhost port which kubectl is forwarding to the git-server Pod.
func forwardToFreePort(t *testing.T, kcfg, pod string) int {
	t.Helper()

	cmd := exec.Command("kubectl", "--kubeconfig", kcfg, "port-forward",
		"-n", testGitNamespace, pod, ":22")

	stdout := &strings.Builder{}
	cmd.Stdout = stdout

	err := cmd.Start()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err := cmd.Process.Kill()
		if err != nil {
			t.Errorf("killing port forward process: %v", err)
		}
	})

	port := 0
	// In CI, 1% of the time this takes longer than 10 seconds, so 20 seconds seems
	// like a reasonable amount of time to wait.
	took, err := Retry(20*time.Second, func() error {
		s := stdout.String()
		if !strings.Contains(s, "\n") {
			return errors.New("nothing written to stdout for kubectl port-forward")
		}

		line := strings.Split(s, "\n")[0]

		// Sample output:
		// Forwarding from 127.0.0.1:44043 -> 22
		_, err = fmt.Sscanf(line, "Forwarding from 127.0.0.1:%d -> 22", &port)
		if err != nil {
			t.Fatalf("unable to parse port-forward output: %q", s)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("took %v to wait for sync", took)

	return port
}
