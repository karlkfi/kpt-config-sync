# Testing

## Unit Tests

To run unit tests and linters:

```console
make
```

## E2E Tests

end-to-end tests deploy GKE Policy Management components on a Kubernetes cluster
from your current context, and then verify functionality through Git commits:

```console
make test-e2e-all
```

## E2E prober tests

Prober tests verify the GCP functionality on a predefined test cluster, using
the tests defined in your local repository, and a predefined prober GCP
service account.

```console
make test-e2e-prober
```

## Working on the e2e framework or e2e tests.

While doing development of e2e test / framework features, it's desirable to skip
steps in the full e2e process. The following commands are available for finer
grained control. This is now supported for -git and -gcp suffixes.

1- Build GKE Policy Management and end to end images. You must do this each time
you make changes to .go code.

```console
make e2e-image-all
```

2- Set up the test environment on your cluster

```console
# git
make test-e2e-dev-git E2E_FLAGS="--setup"

# gcp
make test-e2e-dev-gcp E2E_FLAGS="--setup"
```

3- Run specific test with full debug output. See E2E_FLAGS section for filter
flag usage

```console
# git
make test-e2e-dev-git E2E_FLAGS="--test --tap --test_filter acme"

# gcp
make test-e2e-dev-gcp E2E_FLAGS="--test --tap --test_filter acme"
```

4- Clean up the test environment

```console
# git
make test-e2e-dev-git E2E_FLAGS="--clean"

# gcp
make test-e2e-dev-gcp E2E_FLAGS="--clean"
```

### E2E_FLAGS

Name          | Value                                                                                                                        | Example
------------- | ---------------------------------------------------------------------------------------------------------------------------- | -------
--test_filter | the filter for test casess as a regex                                                                                        | The following filters for a test containing 'backend' E2E_FLAGS="--test_filter backend"
--file_filter | the filter for test files as a regex                                                                                         | The following filters for a file containing 'acme-foo' E2E_FLAGS="--file_filter acme-foo"
--preclean    | boolean, uninstalls GKE Policy Management prior to setup/test, useful for making a 'clean slate' without doing anything else | E2E_FLAGS="--preclean"
--clean       | boolean, uninstalls GKE Policy Management and test infra from cluster at end of execution                                    | E2E_FLAGS="--clean"
--setup       | boolean, sets up GKE Policy Management and test infra on cluster                                                             | E2E_FLAGS="--setup"
--tap         | boolean, emit tap output while tests are running, useful for debugging                                                       | E2E_FLAGS="--tap"
--test        | boolean, run e2e tests                                                                                                       | E2E_FLAGS="--test"
