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
HOSTNAME=localhost
PORTNAME=10443
cat > $HOME/.minikube/addons/webhook.kubeconfig << EOF
clusters:
  - name: authorizer
    cluster:
      certificate-authority: /var/lib/localkube/certs/ca.crt
      server: https://${HOSTNAME}:${PORTNAME}/authorize
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
chmod ug+x $HOME/.minikube/addons/authorizer

cp minikube/bootlocal.sh $HOME/.minikube/addons/bootlocal.sh
chmod ug+x $HOME/.minikube/addons/bootlocal.sh

cat > $HOME/.minikube/addons/authorizer.service << EOF
[Unit]
Description=Demo Webhook Authorizer for Kubernetes
Documentation=http://www.example.com

[Service]
Type=notify
Restart=always
RestartSec=3

ExecStart=/etc/kubernetes/addons/authorizer --notify_systemd --logtostderr --vmodule=main=2 --listen_hostport=:${PORTNAME} --cert_file=/etc/kubernetes/addons/server.crt --server_key=/etc/kubernetes/addons/server.key

ExecReload=/bin/kill -s HUP \$MAINPID

[Install]
WantedBy=localkube.service

EOF
