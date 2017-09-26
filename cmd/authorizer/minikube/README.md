This directory contains deployment-specific files for minikube.

The scripts here should not be invoked directly, but rather through an
appropriate Makefile target in the directory above.

* `init.sh`: Copies the state needed for correct minikube startup with our
  additions hooked in.
* `start.sh`: A wrapper around `minikube start` that passes the flags required
  for webhook.   You should not need to invoke it directly though.  Use the
  Makefile target `minikube_start` instead.  It expects that an authorizer
  server is already running.
* `gencert.sh`: Creates the public and private keys to be used for the
  authorizer, and the server certificate, signed by the same certificate
  authority that was generated for the minikube cluster.

