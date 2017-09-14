This is a sample webhook authorizer for k8s.

It always allows access to any object it is asked to authorize.

# Prerequisites

* Kubernetes installation and source is required as a core dependency.

* Minikube is required for running local development setup.

* Docker command line utility is required for interacting with Minikube's docker
  daemon.  Running an actual docker daemon is not required.

* Go version 1.9 is required for HTTPS testing fixture.

* OpenSSL is required for key generation.

* GNU make is used for building non-golang targets.  Any recent version will do.

# Building

Build, test and create a minimalistic docker image for the server:

```
make docker_minikube
```

# Running the setup

This assumes that Minikube is not running. Open two terminal windows.

Run in one:

```
make debug_authorizer_start
```

Run in the other:

```
make minikube_start
```

This should run the Minikube-with-webhook setup that calls back to your machine.

# Cleanup

This command will remove all compilation artifacts.  It will not remove any
docker images:

```
make clean
```

This commands will remove the docker image of the sample server:

```
make docker_clean
```

# Starting as a pod

For now, this is a manual step.  This command will create and run a pod
that contains a single container with the sample server.

```
kubectl create -f pod.yaml
```

# Cleanup

This command stops the server and removes the pod.

```
kubectl delete -f pod.yaml
```

# Convenience targets

These build targets are defined to automate state setup for development.

* `make debug_authorizer_start`: runs an authorizer copy on your machine.  This
  is be used to provide a target for the apiserver webhook to call into.  Make
  sure that the authorizer is running before you start minikube.

* `make minikube_start`: starts Minikube which is configured to use the webhook
  authorizer running on the host machine.

* `make minikube_stop`: stops a Minikube.  This is mostly for symmetry with
  `minikube_start`.

