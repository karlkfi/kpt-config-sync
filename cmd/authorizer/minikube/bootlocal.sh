#! /bin/sh
# This script starts the authorizer on the master machine.

MINIKUBE_HOST_ADDRESS=${MINIKUBE_HOST_ADDRESS:-localhost}
APISERVER_PORT=8433
PORTNUMBER=10443

chmod ug+x /etc/kubernetes/addons/authorizer &> /var/log/bootlocal.sh.log
cp /etc/kubernetes/addons/authorizer.service /lib/systemd/system &> /var/log/bootlocal.sh.log

cat > /lib/systemd/system/authorizer.service << EOF
[Unit]
Description=Demo Webhook Authorizer for Kubernetes
Documentation=http://www.example.com

[Service]
Type=notify
Restart=always
RestartSec=3

ExecStart=/etc/kubernetes/addons/authorizer       \
  --notify_systemd --logtostderr --vmodule=main=2 \
  --listen_hostport=:${PORTNUMBER}                  \
  --cert_file=/etc/kubernetes/addons/server.crt   \
  --server_key=/etc/kubernetes/addons/server.key  \
  --ca_cert_file=/var/lib/localkube/certs/ca.crt  \
  --apiserver_hostport=${MINIKUBE_HOST_ADDRESS}:${APISERVER_PORT}

ExecReload=/bin/kill -s HUP \$MAINPID

[Install]
WantedBy=localkube.service

EOF

# Manage the authorizer process using systemd.  This loads the newly-created
# systemd configuration and restarts the daemon.
systemctl daemon-reload
systemctl enable authorizer.service
systemctl restart authorizer.service || true

