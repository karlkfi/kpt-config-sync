# Testing

## Unit Tests

During iterative development, run unit tests and linters using:

```console
make
```

## E2E Tests

end-to-end tests deploy GKE Policy Management components on a Kubernetes cluster
from your current context, and then verify functionality through Git commits.
Running the tests requires local kubeconfig set up properly with Nomos cluster;
the cluster's service account needs storage.objectViewer role on the GCP project
that holds Docker images in Google Container Registry.

```console
make test-e2e-git
```

## Local nomos vet tests

To run nomos vet locally on example repos, run:

```console
make test-nomos-vet-local
```

This requires a Nomos cluster configured and in your kubeconfig context.

## e2e test filters

It's possible to filter which e2e tests are run, rather than running the whole suite.

You can restrict by test name:
```console
make test-e2e-git E2E_FLAGS="--test_filter acme"
```

Or by file name:
```console
make test-e2e-git E2E_FLAGS="--file_filter acme"
```

## E2E Tests with a custom Operator

The e2e tests install Nomos using the Nomos operator, the code for which lives in the
[nomos-operator
repo](https://gke-internal.git.corp.google.com/cluster-lifecycle/cluster-operators/).
By default, e2e tests run against the latest release of the operator. However,
you can run e2e tests against your own build of the operator by doing the
following:

Check out the `cluster-operators` repo. Instructions can be found in
[the nomos-operator readme](https://gke-internal.git.corp.google.com/cluster-lifecycle/cluster-operators/+/master/nomos-operator/README.md#clone-the-git-repo)

Once you have made the changes you wish to test in that repository, run

```console
make release-user
```

This pushes your repo's version of the nomos-operator to a user-private
location in GCR.

Then, return to the main nomos repo and run tests with the `-user` target:
```console
make test-e2e-git-user E2E_FLAGS="--file_filter acme"
```

## Isolating setup, tests, and cleanup.

The e2e test suite starts with a suite setup, then runs tests, and finally does a cleanup.
(The setup also contains a clean step.) You can run these stages individually using the
`test-e2e-dev-git` target.

1- Build GKE Policy Management and end to end images. You must do this each time
you make changes to .go code.

```console
make e2e-image-all
```

2- Set up the test environment on your cluster

```console
# git
make test-e2e-dev-git E2E_FLAGS="--setup"
```

3- Run tests with `--test`. This example runs a filtered set of tests with full debug output.

```console
# git
make test-e2e-dev-git E2E_FLAGS="--test --tap --test_filter acme"
```

4- Clean up the test environment

```console
# git
make test-e2e-dev-git E2E_FLAGS="--clean"
```

### E2E_FLAGS

Name          | Value                                                                                                                        | Example
------------- | ---------------------------------------------------------------------------------------------------------------------------- | -------
--clean       | boolean, uninstalls GKE Policy Management and test infra from cluster at end of execution                                    | E2E_FLAGS="--clean"
--file_filter | the filter for test files as a regex                                                                                         | The following filters for a file containing 'acme-foo' E2E_FLAGS="--file_filter acme-foo"
--preclean    | boolean, uninstalls GKE Policy Management prior to setup/test, useful for making a 'clean slate' without doing anything else | E2E_FLAGS="--preclean"
--setup       | boolean, sets up GKE Policy Management and test infra on cluster                                                             | E2E_FLAGS="--setup"
--tap         | boolean, emit tap output while tests are running, useful for debugging                                                       | E2E_FLAGS="--tap"
--test        | boolean, run e2e tests                                                                                                       | E2E_FLAGS="--test"
--test_filter | the filter for test cases as a regex                                                                                         | The following filters for a test containing 'backend' E2E_FLAGS="--test_filter backend"
--timing      | boolean, print timing info for each test. Also turns on --tap.                                                               | E2E_FLAGS="--timing"
