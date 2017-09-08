Some random policy node CRDs you can add to a cluster

First, define the CRD in your cluster
```
kubectl create -f manifests/policy-node-crd.yaml
```

Then, add the sample CRDs from this folder
```
kubectl create -f policyNodeOU1.yaml
kubectl create -f policyNodeNS1.yaml
kubectl create -f policyNodeNS2.yaml

kubectl get policynodes
```
