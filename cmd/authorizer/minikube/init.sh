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
WEBHOOK_HOSTNAME=${WEBHOOK_HOSTNAME:-10.0.0.112}
PORTNUMBER=443
cat > $HOME/.minikube/addons/webhook.kubeconfig << EOF
clusters:
  - name: authorizer
    cluster:
      certificate-authority: /var/lib/localkube/certs/ca.crt
      server: https://${WEBHOOK_HOSTNAME}:${PORTNUMBER}/authorize
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

cp server.key $HOME/.minikube/addons/server.key
cp server.crt $HOME/.minikube/addons/server.crt
cp authorizer $HOME/.minikube/addons/authorizer
cp minikube/bootlocal.sh $HOME/.minikube/addons/bootlocal.sh

