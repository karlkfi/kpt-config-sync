package e2e

import (
	"fmt"
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSyncingThroughAProxy(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo)

	nt.T.Logf("Set up the tiny proxy service and Override the RootSync object with proxy setting")
	nt.MustKubectl("apply", "-f", "../testdata/proxy")
	nt.T.Cleanup(func() {
		nt.MustKubectl("delete", "-f", "../testdata/proxy")
		_, err := nomostest.Retry(nt.DefaultWaitTimeout, func() error {
			return nt.ValidateNotFound("tinyproxy-deployment", "proxy-test", &appsv1.Deployment{})
		})
		if err != nil {
			nt.T.Fatal(err)
		}
	})
	_, err := nomostest.Retry(nt.DefaultWaitTimeout, func() error {
		return nt.Validate("tinyproxy-deployment", "proxy-test", &appsv1.Deployment{}, hasReadyReplicas(1))
	})
	if err != nil {
		nt.T.Fatal(err)
	}
	nt.T.Log("Verify the NoOpProxyError")
	nt.WaitForStalledError("Validation", `KNV1061: RootSyncs which declare spec.git.proxy must declare spec.git.auth="none", "cookiefile" or "token"`)

	rs := fake.RootSyncObjectV1Beta1()
	nt.T.Log("Set auth type to cookiefile")
	nt.MustMergePatch(rs, `{"spec": {"git": {"auth": "cookiefile"}}}`)
	nt.T.Log("Verify the secretRef error")
	nt.WaitForStalledError("Secret", `git secretType was set as "cookiefile" but cookie_file key is not present`)

	nt.T.Log("Set auth type to token")
	nt.MustMergePatch(rs, `{"spec": {"git": {"auth": "token"}}}`)
	nt.T.Log("Verify the secretRef error")
	nt.WaitForStalledError("Secret", `git secretType was set as "token" but token key is not present`)

	nt.T.Log("Set auth type to none")
	nt.MustMergePatch(rs, `{"spec": {"git": {"auth": "none", "secretRef": {"name":""}}}}`)
	nt.T.Log("Verify no errors")
	rs = &v1beta1.RootSync{}
	if err = nt.Get("root-sync", configmanagement.ControllerNamespace, rs); err != nil {
		nt.T.Fatal(err)
	}
	sha1Fn := func(nt *nomostest.NT) (string, error) {
		rs = &v1beta1.RootSync{}
		if err = nt.Get("root-sync", configmanagement.ControllerNamespace, rs); err != nil {
			return "", err
		}
		return rs.Status.LastSyncedCommit, nil
	}
	nt.WaitForRepoSyncs(nomostest.WithRootSha1Func(sha1Fn), nomostest.WithSyncDirectory("foo-corp"))
}

func hasReadyReplicas(replicas int32) nomostest.Predicate {
	return func(o client.Object) error {
		deployment, ok := o.(*appsv1.Deployment)
		if !ok {
			return nomostest.WrongTypeErr(deployment, &appsv1.Deployment{})
		}
		actual := deployment.Status.ReadyReplicas
		if replicas != actual {
			return fmt.Errorf("expected %d ready replicas, but got %d", replicas, actual)
		}
		return nil
	}
}
