#! /bin/sh
# This script starts the authorizer on the master machine.

chmod ug+x /etc/kubernetes/addons/authorizer &> /var/log/bootlocal.sh.log
cp /etc/kubernetes/addons/authorizer.service /lib/systemd/system &> /var/log/bootlocal.sh.log

# Manage the authorizer process using systemd.  This loads the newly-created
# systemd configuration and restarts the daemon.
systemctl daemon-reload
systemctl enable authorizer.service
systemctl restart authorizer.service || true

