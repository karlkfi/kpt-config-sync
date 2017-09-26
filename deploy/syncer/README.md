
# Syncer Service

## Building

Create a syncer:test image locally
```bash
make
```

Create a syncer:test image in minikube's repo:
```
make-minikube.sh
```

## Creating service account

```
./create-service-account.sh
```

## Starting syncer service pod

```
kubectl create -f syncer-pod.yaml
```
