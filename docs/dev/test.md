# Testing

## Unit Tests

To run unit tests and linters:

```console
make
```

## E2E Tests

end-to-end tests deploy Nomos components on a Kubernetes cluster and verify
functionality through Git commits:

```console
make test-e2e-all
```

If debugging something and want to prevent cleanup after tests runs:

```console
make test-e2e-nocleanup-{git,gcp}
```

During iterative development of e2e tests, you may want to skip time-consuming
setup steps:

1- Run tests without cleanup once:

```console
 make test-e2e-nocleanup-{git,gcp}
```

2- Make changes to tests and run:

```console
  make test-e2e-nosetup-{git,gcp}
```

3- Repeat step 2 as necessary.

## Working on the e2e framework or e2e tests.

While doing development of e2e test / framework features, it's desirable to skip
steps in the full e2e process. The following commands are available for finer
grained control.

1- Build nomos and end to end images

```console
make e2e-image-all
```

2- Set up the test environment on your cluster

```console
make test-e2e-dev-git E2E_FLAGS="--setup"
```

3- Run specific test with full debug output. See E2E_FLAGS section for filter
flag usage

```console
make test-e2e-dev-git E2E_FLAGS="--test --tap --filter acme/acme"
```

4- Clean up the test environment

```console
make test-e2e-dev-git E2E_FLAGS="--clean"
```

### E2E_FLAGS

Name     | Value                                                                  | Example
-------- | ---------------------------------------------------------------------- | -------
--filter | the filter for tests, formatted as [file pattern] '/' [test pattern]   | The following filters for a file containing 'acme-foo' with a test containing 'backend' E2E_FLAGS="--filter acme-foo/backend"
--clean  | boolean, uninstalls nomos and test infra from cluster                  | E2E_FLAGS="--clean"
--setup  | boolean, sets up nomos and test infra on cluster                       | E2E_FLAGS="--setup"
--tap    | boolean, emit tap output while tests are running, useful for debugging | E2E_FLAGS="--tap"
--test   | boolean, run e2e tests                                                 | E2E_FLAGS="--test"
