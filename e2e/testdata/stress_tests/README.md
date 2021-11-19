The testdata in this directory are used for the stress tests in
`e2e/testcases/stress_test.go`.

`configs-5000-cms-1-ns`: includes one namespace `my-ns-1`, and 5000 configmaps under the
namespace (`cm-1` - `cm-5000`).

`configs-1-crd-1000-ns`: includes one CronTab CRD, and 1000 namespaces (`foo1` - `foo1000`). Every namespace includes a
ConfigMap object and a CronTab CR.
