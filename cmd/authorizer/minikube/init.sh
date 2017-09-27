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
WEBHOOK_ADDRESS=${WEBHOOK_ADDRESS:-10.0.0.112}
PORTNUMBER=443

# For example "gcr.io/yourproject/"
DOCKER_IMAGE_REGISTRY=${DOCKER_IMAGE_REGISTRY:-""}

cat > $HOME/.minikube/addons/webhook.kubeconfig << EOF
kind: Config
preferences: {}
clusters:
  - name: authorizer
    cluster:
      certificate-authority: /var/lib/localkube/certs/ca.crt
      server: https://${WEBHOOK_ADDRESS}:${PORTNUMBER}/authorize
users:
  - name: apiserver
    user:
      client-certificate: /var/lib/localkube/certs/apiserver.crt
      client-key: /var/lib/localkube/certs/apiserver.key
current-context: webhook
contexts:
  - context:
      cluster: authorizer
      user: apiserver
    name: webhook
EOF

cat > minikube/authorizer_deploy.yaml << EOF
apiVersion: v1
kind: Service
metadata:
  name: authorizer
spec:
  selector:
    app: authz
  ports:
  - name: foo
    port: 443
    targetPort: 8443
  clusterIP: ${WEBHOOK_ADDRESS}
---
apiVersion: v1
kind: Pod
metadata:
  name: authorizer
  namespace: default
  labels:
    app: authz
spec:
  containers:
  - name: authorizer
    image: ${DOCKER_IMAGE_REGISTRY}authorizer:experimental
    ports:
      - containerPort: 8443
    args: [
      "--logtostderr",
      "--vmodule=main=2"
    ]
  restartPolicy: Always
EOF
