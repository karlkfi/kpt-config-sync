# Testing

## Unit Tests

During iterative development, run unit tests and linters using:

```console
make
```

## E2E Tests

end-to-end tests deploy Anthos Config Management components on a Kubernetes
cluster from your current context, and then verify functionality through Git
commits. Running the tests requires local kubeconfig set up properly with Nomos
cluster; the cluster's service account needs storage.objectViewer role on the
GCP project that holds Docker images in Google Container Registry.

```console
make test-e2e
```

Further information is now in g3docs, at go/acm-bats-test-guide.

## Local nomos vet tests

To run nomos vet locally on example repos, run:

```console
make test-nomos-vet-local
```

This requires a Nomos cluster configured and in your kubeconfig context.
