package resource_quota

import (
	"testing"
	"time"

	"github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/meta/fake"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestFullSync(t *testing.T) {
	// Limits and structure
	policyNodes := []runtime.Object{
		makePolicyNode("animals", "", core_v1.ResourceList{
			"bones": resource.MustParse("10"),
		}, true),
		makePolicyNode("kittiesandponies", "animals", core_v1.ResourceList{
			"hay":  resource.MustParse("10"),
			"milk": resource.MustParse("5"),
		}, true),
		makePolicyNode("kitties", "kittiesandponies", core_v1.ResourceList{
			"hay": resource.MustParse("5"),
		}, false),
		makePolicyNode("ponies", "kittiesandponies", core_v1.ResourceList{
			"hay":  resource.MustParse("15"),
			"milk": resource.MustParse("5"),
		}, false),
		makePolicyNode("dinosaurs", "animals", core_v1.ResourceList{}, true),
		makePolicyNode("raptors", "dinosaurs", core_v1.ResourceList{}, false),
		makePolicyNode("turtles", "animals", core_v1.ResourceList{}, false),
	}

	// Quotas at leaf levels
	quotas := []runtime.Object{
		makeResourceQuota("kitties", core_v1.ResourceList{
			"hay": resource.MustParse("2"),
		}),
		makeResourceQuota("ponies", core_v1.ResourceList{
			"hay":  resource.MustParse("2"),
			"milk": resource.MustParse("2"),
		}),
		makeResourceQuota("turtles", core_v1.ResourceList{
			"cabbage": resource.MustParse("1"),
		}),
	}

	// starting stolos quotas
	stolosQuotas := []runtime.Object{
		// This one should get cleared out
		makeStolosQuota("dinosaurs", core_v1.ResourceQuotaStatus{
			Used: core_v1.ResourceList{
				"hay": resource.MustParse("2"),
			},
			Hard: core_v1.ResourceList{
				"hay": resource.MustParse("2"),
			},
		}),
		// This should get modified
		makeStolosQuota("kittiesandponies", core_v1.ResourceQuotaStatus{
			Used: core_v1.ResourceList{
				"hay": resource.MustParse("2"),
			},
		}),
		// animals should get created fresh
	}

	// This is the state of the world we want to see at the end of a full sync of the controller.
	// This tests creates, modifies and clears out both used and hard.
	expectedQuotas := map[string]v1.StolosResourceQuotaSpec{
		"dinosaurs": {},
		"kittiesandponies": {Status: core_v1.ResourceQuotaStatus{
			Hard: core_v1.ResourceList{
				"hay":  resource.MustParse("10"),
				"milk": resource.MustParse("5"),
			},
			Used: core_v1.ResourceList{
				"hay":  resource.MustParse("4"), // 2 + 2 from kitties and ponies each
				"milk": resource.MustParse("2"),
			}},
		},
		"animals": {Status: core_v1.ResourceQuotaStatus{
			Hard: core_v1.ResourceList{
				"bones": resource.MustParse("10"),
			},
			Used: core_v1.ResourceList{
				"hay":     resource.MustParse("4"),
				"milk":    resource.MustParse("2"),
				"cabbage": resource.MustParse("1"),
			}},
		},
	}

	fakeClient := fake.NewClientWithData(quotas, append(policyNodes, stolosQuotas...))

	// Run the controller!
	controller := NewController(fakeClient, nil)

	controller.Run()
	// Controller is async so we wait a second to let it do its thing.
	time.Sleep(time.Second)

	// Now ensure the state of the world is as expected
	for namespace, expectedQuotaStatus := range expectedQuotas {
		actualQuota, err := fakeClient.PolicyHierarchy().K8usV1().StolosResourceQuotas(namespace).Get(ResourceQuotaObjectName, meta_v1.GetOptions{})
		if err != nil {
			t.Errorf("Unexpected error %s", err)
		}

		if !specEqual(actualQuota.Spec, expectedQuotaStatus) {
			t.Errorf("Expected quota spec for namespace %s:\n%v \n but got:\n%v", namespace, expectedQuotaStatus, actualQuota.Spec)
		}
	}
}

func makeStolosQuota(namespace string, status core_v1.ResourceQuotaStatus) *v1.StolosResourceQuota {
	return &v1.StolosResourceQuota{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      ResourceQuotaObjectName,
			Namespace: namespace,
			Labels:    PolicySpaceQuotaLabels,
		},
		Spec: v1.StolosResourceQuotaSpec{Status: status},
	}
}
