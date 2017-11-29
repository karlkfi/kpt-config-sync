# stolos-kube-up.sh

This is a script that creates a Kubernetes cluster off of Stolos-patched
Kubernetes on GCE.

To install your Kubernetes cluster, make sure your cluster is DOWN (i.e. not
running).

Then download the [K8S GCE installation script][1] from k8s.io.  Place it into
`$HOM/local/opt/k8s`.  This is the default setting.  Feel free to look into the
script and provide a different directory that you like better.

Then run, from stolos home directory:

   ./tools/stolos-kube-up.sh

This will install the Stolos-aware Kubernetes cluster that you can continue
using as usual.  The only difference will be that it will be ready to run
the webhook authorizer as a pod.

[1]: https://kubernetes.io/docs/getting-started-guides/gce

