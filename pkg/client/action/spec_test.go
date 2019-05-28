package action

import (
	"reflect"
	"sort"
	"testing"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

func fakeEqual(_ runtime.Object, _ runtime.Object) bool {
	return true
}

func TestSpecListNamespaced(t *testing.T) {
	roles := []*rbacv1.Role{
		{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "role-1"}},
		{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "role-2"}},
		{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "role-3"}},
	}
	var objs []runtime.Object
	for _, role := range roles {
		objs = append(objs, role)
	}

	client := fake.NewSimpleClientset(objs...)
	factory := informers.NewSharedInformerFactory(client, time.Second*10)
	lister := factory.Rbac().V1().Roles().Lister()
	roleSpec := NewSpec(&rbacv1.Role{}, rbacv1.SchemeGroupVersion, fakeEqual, client.RbacV1(), lister)

	factory.Start(nil)
	factory.WaitForCacheSync(nil)

	listObjs, err := roleSpec.List("ns", labels.Everything())
	if err != nil {
		t.Errorf("Failed to list roles")
	}
	var names []string
	for _, obj := range listObjs {
		names = append(names, obj.(metav1.Object).GetName())
	}
	sort.Strings(names)
	expect := []string{"role-1", "role-2", "role-3"}
	if !reflect.DeepEqual(names, expect) {
		t.Errorf("Did not list correct names, expected %s, got %s %v", expect, names, listObjs)
	}
}

func TestSpecListCluster(t *testing.T) {
	clusterRoles := []*rbacv1.ClusterRole{
		{ObjectMeta: metav1.ObjectMeta{Name: "cluster-1"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "cluster-2"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "cluster-3"}},
	}
	var objs []runtime.Object
	for _, clusterRole := range clusterRoles {
		objs = append(objs, clusterRole)
	}
	client := fake.NewSimpleClientset(objs...)
	factory := informers.NewSharedInformerFactory(client, time.Second*10)
	lister := factory.Rbac().V1().ClusterRoles().Lister()
	clusterRoleSpec := NewSpec(&rbacv1.ClusterRole{}, rbacv1.SchemeGroupVersion, fakeEqual, client.RbacV1(), lister)

	factory.Start(nil)
	factory.WaitForCacheSync(nil)

	listObjs, err := clusterRoleSpec.List("", labels.Everything())
	if err != nil {
		t.Errorf("Failed to list ClusterRoles")
	}
	var names []string
	for _, obj := range listObjs {
		names = append(names, obj.(metav1.Object).GetName())
	}
	sort.Strings(names)
	expect := []string{"cluster-1", "cluster-2", "cluster-3"}
	if !reflect.DeepEqual(names, expect) {
		t.Errorf("Did not list correct names, expected %s, got %s %v", expect, names, listObjs)
	}
}
