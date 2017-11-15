This is a webhook admission controller for Kubernetes.
This [is/will eventually be] a hierarchical Resource Quota admission controller that ensures quota policy
is enforced along the hierarchy of namespaces of the kubernetes cluster in which it is configured.

# Setup

You need a customized 1.8 or later Kubernetes installation.

After installing Kubernetes, add the following arguments to the api server startup
(located at /etc/kubernetes/manifests/kube-apiserver.manifest on the master or by modifying kube-up.sh)

* Add GenericAdmissionWebhook to the flag: --admission-control=...
* Add the flag: --runtime-config=admissionregistration.k8s.io/v1alpha1

# Building

export GCP_PROJECT=<my GCP project id>

```
make certs
```
Will generate certificates for the service. Must be done before deploy! Generally should only be done once.

```
make deploy
```
Will build a docker image, push it to GCR in your project and start a pod using the image with the admission controller.
This should be done every time the code changes.

```
