#!/bin/bash
#
# Sets up the webhook authorizer configuration for minikube.

set -v

# The file format below is the "kubeconfig" file format.  This is used
# to configure "system" parts of Kubernetes.
# Details are here: https://kubernetes.io/docs/admin/authorization/webhook/
# While it has not been called out explicitly, it seems that the webhook config
# is just reusing the file format and repurposing it to configure a webhook.
# This may explain the non-intuitive names of the fields in the config file.
HOSTNAME=$(hostname)
cat > $HOME/.minikube/addons/webhook.kubeconfig << EOF
clusters:
  - name: authorizer
    cluster:
      certificate-authority: /var/lib/localkube/certs/ca.crt
      server: https://$HOSTNAME:8443/authorize
users:
  - name: minikube
    user:
      client-certificate: /var/lib/localkube/certs/apiserver.crt
      client-key: /var/lib/localkube/certs/apiserver.key
current-context: webhook
contexts:
- context:
    cluster: authorizer
    user: minikube
  name: webhook
EOF

