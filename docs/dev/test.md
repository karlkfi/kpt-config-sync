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

#### Options

Env Var     | Value                                                                                                                                                                                                   | Example
----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------
TEST_FILTER | regex for filtering test files and testcases from the e2e/testcases directory. The '/' character delimits the [file]/[testcase] patterns. If no '/' is present, the filter will apply to testcases only | make TEST_FILTER=namespaces/create test-e2e
