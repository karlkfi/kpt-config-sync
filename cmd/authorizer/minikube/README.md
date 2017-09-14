This directory contains deployment-specific files for
minikube.

The scripts here should not be invoked directly, but rather through an
appropriate Makefile target in the directory above.

* `init.sh`: Copies the state needed for correct minikube startup with our
  additions hooked in.
* `start.sh`: A wrapper around `minikube start` that passes the flags required
  for webhook.   You should not need to invoke it directly though.  Use the
  Makefile target `minikube_start` instead.  It expects that an authorizer
  server is already running.
* `webhook.kubeconfig`: the kubeconfig-formatted file used to declare the
  webhook auth configuration of the apiserver.

`webhook.kubeconfig` is currently set up to look for the webhook authorizer on
the minikube host machine.  This is not the ideal, nor the final setup, but is
useful for the time being as we're bootstrapping the work here.

