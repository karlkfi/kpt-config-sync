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

## Local nomos vet tests

To run nomos vet locally on example repos, run:

```console
make test-nomos-vet-local
```

This requires a Nomos cluster configured and in your kubeconfig context.

## e2e test filters

It's possible to filter which e2e tests are run, rather than running the whole
suite.

You can restrict by test name:
```console
make test-e2e E2E_FLAGS="--test_filter acme"
```

Or by file name:
```console
make test-e2e E2E_FLAGS="--file_filter acme"
```

Also, It is possible to ignore a test while running the make target.

You can ignore a test by test name:
```console
make test-e2e E2E_FLAGS="--file_ignore_filter acme.bats"
```

## E2E Tests with a custom Operator

This functionality has been removed in an effort to decouple nomos from the operator.

In general, if a test needs the operator to verify some functionality, that test should
be added to the operator repo.

## Isolating setup, tests, and cleanup.

The e2e test suite starts with a suite setup, then runs tests, and finally does
a cleanup. (The setup also contains a clean step.) You can run these stages
individually using the `test-e2e-dev` target.

1- Build Anthos Config Management and end to end images. You must do this
each time you make changes to .go code.

```console
make e2e-image-all
```

2- Set up the test environment on your cluster

```console
# git
make test-e2e-dev E2E_FLAGS="--setup"
```

3- Run tests with `--test`. This example runs a filtered set of tests with full
debug output.

```console
# git
make test-e2e-dev E2E_FLAGS="--test --tap --test_filter acme"
```

4- Clean up the test environment

```console
# git
make test-e2e-dev E2E_FLAGS="--clean"
```

### E2E_FLAGS

Name          | Value                                                                                                                               | Example
------------- | ----------------------------------------------------------------------------------------------------------------------------------- | -------
--clean       | boolean, uninstalls Anthos Config Management and test infra from cluster at end of execution                                    | E2E_FLAGS="--clean"
--file_filter | the filter for test files as a regex                                                                                                | The following filters for a file containing 'acme-foo' E2E_FLAGS="--file_filter acme-foo"
--preclean    | boolean, uninstalls Anthos Config Management prior to setup/test, useful for making a 'clean slate' without doing anything else | E2E_FLAGS="--preclean"
--setup       | boolean, sets up Anthos Config Management and test infra on cluster                                                             | E2E_FLAGS="--setup"
--tap         | boolean, emit tap output while tests are running, useful for debugging                                                              | E2E_FLAGS="--tap"
--test        | boolean, run e2e tests                                                                                                              | E2E_FLAGS="--test"
--test_filter | the filter for test cases as a regex                                                                                                | The following filters for a test containing 'backend' E2E_FLAGS="--test_filter backend"
--timing      | boolean, print timing info for each test. Also turns on --tap.                                                                      | E2E_FLAGS="--timing"
--file_ignore_filter | the filter for test files as a regex                                                                                     | E2E_FLAGS="--file_ignore_filter acme"
