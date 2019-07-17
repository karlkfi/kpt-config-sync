package admissioncontroller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/clientgen/apis"
	namespaceconfigversions "github.com/google/nomos/clientgen/informer"
	informersv1 "github.com/google/nomos/clientgen/informer/configmanagement/v1"
	"github.com/google/nomos/pkg/service"
	"github.com/pkg/errors"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionregv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// WaitForEndpoint waits for endpoint to come up.
// We have to wait for the endpoint to come up before self registering the webhook,
// otherwise the endpoint will never come up since the admission controller will appear down and
// block all requests, including the endpoint initialization
func WaitForEndpoint(clientset *kubernetes.Clientset, name, namespace string, timeout time.Duration) error {
	for t := time.Now(); time.Since(t) < timeout; time.Sleep(time.Second) {
		endpoint, err := clientset.CoreV1().Endpoints(namespace).Get(name, metav1.GetOptions{})

		if apierrors.IsNotFound(err) {
			glog.Info("Endpoint not ready yet...")
			continue
		}

		if err != nil {
			return errors.Wrap(err, "Failed while checking endpoint readiness")
		}

		if len(endpoint.Subsets) == 0 || len(endpoint.Subsets[0].Addresses) == 0 {
			glog.Info("Endpoint address not ready yet...")
			continue
		} else {
			glog.V(3).Info("Endpoint ready: ", endpoint)
			return nil
		}
	}
	return fmt.Errorf("timed out waiting for endpoint")
}

// GetAPIServerCert retrieves the client certificate used to sign requests from api-server.
//
// See --proxy-client-cert-file flag description:
// https://kubernetes.io/docs/admin/kube-apiserver
func GetAPIServerCert(clientset *kubernetes.Clientset) ([]byte, error) {
	c, err := clientset.CoreV1().ConfigMaps("kube-system").Get("extension-apiserver-authentication", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get client auth from configmap: %v", err)
	}

	pem, ok := c.Data["requestheader-client-ca-file"]
	if !ok {
		return nil, fmt.Errorf("cannot find the ca.crt in the configmap, configMap.Data is %#v", c.Data)
	}
	return []byte(pem), nil
}

// GetWebhookCert returns the contents of the certificate file.
func GetWebhookCert(caCertFile string) ([]byte, error) {
	caCert, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read ca bundle file %q", caCertFile)
	}
	return caCert, nil
}

// RegisterWebhook upserts the webhook configuration with the provided config.
func RegisterWebhook(clientset *kubernetes.Clientset, webhookConfig *admissionregv1beta1.ValidatingWebhookConfiguration) error {
	client := clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations()

	existing, err := client.Get(webhookConfig.Name, metav1.GetOptions{})
	if err == nil {
		glog.Infof("Updating existing ValidatingWebhookConfiguration")
		webhookConfig.ResourceVersion = existing.ResourceVersion
		if _, err2 := client.Update(webhookConfig); err2 != nil {
			return errors.Wrap(err2, "failed to update ValidatingWebhookConfiguration")
		}
	} else if apierrors.IsNotFound(err) {
		glog.Infof("Creating ValidatingWebhookConfiguration")
		if _, err2 := client.Create(webhookConfig); err2 != nil {
			return errors.Wrap(err2, "failed to create ValidatingWebhookConfiguration")
		}
	} else {
		return errors.Wrap(err, "failed retrieving existing ValidatingWebhookConfiguration")
	}
	return nil
}

// Serve returns a handler function for the Admission controller.
func Serve(controller Admitter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body []byte
		if r.Body == nil {
			glog.Error("empty request")
			http.Error(w, "empty request", http.StatusBadRequest)
			return
		}

		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}

		// verify the content type is accurate
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			glog.Errorf("contentType=%s, expect application/json", contentType)
			http.Error(w, fmt.Sprintf("Invalid content type %s", contentType), http.StatusBadRequest)
			return
		}

		review := admissionv1beta1.AdmissionReview{}
		if err := json.Unmarshal(body, &review); err != nil {
			glog.Error(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		reviewStatus := controller.Admit(review)
		glog.Infof("Admission decision for namespace %s, object %s.%s %s: %v",
			review.Request.Namespace, review.Request.Kind.Kind, review.Request.Kind.Version, review.Request.Name, reviewStatus)
		ar := admissionv1beta1.AdmissionReview{
			Response: reviewStatus,
		}

		resp, err := json.Marshal(ar)
		if err != nil {
			glog.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(resp); err != nil {
			glog.Error(err)
		}
	}
}

// ServeTLS calls server.serveTLS
func ServeTLS(server *http.Server, listener net.Listener, stopChannel chan struct{}) {
	glog.Info("Server listening on ", server.Addr)
	err := server.ServeTLS(listener.(*net.TCPListener), "", "")

	if err != nil {
		glog.Fatal("Failed during serveTLS: ", err)
	}
	close(stopChannel)
}

// ServeFunc returns the serving function for this server.  Use for testing.
func ServeFunc(controller Admitter) http.HandlerFunc {
	return service.WithStrictTransport(service.NoCache(Serve(controller)))
}

// SetupHierarchicalQuotaInformer returns a newly configured HierarchicalQuotaInformer.
func SetupHierarchicalQuotaInformer(config *rest.Config) (informersv1.HierarchicalQuotaInformer, error) {
	configManagementClient, err := apis.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create configmanagement client")
	}

	namespaceConfigFactory := namespaceconfigversions.NewSharedInformerFactory(
		configManagementClient, time.Minute,
	)

	hierarchicalQuotaInformer := namespaceConfigFactory.Configmanagement().V1().HierarchicalQuotas()
	hierarchicalQuotaInformer.Informer()
	namespaceConfigFactory.Start(nil)

	return hierarchicalQuotaInformer, nil
}
