This is a sample webhook authorizer for k8s.

It always allows access to any object it is asked to authorize.

# Prerequisites

For now, this example requires an up and running `minikube`, an installed
`docker` utility for building the images, and an installed `kubectl` utility.

Go version 1.9 is required.

GNU make is used.

# Building

Build, test and create a minimalistic docker image for the server:

```
make docker_minikube
```

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

