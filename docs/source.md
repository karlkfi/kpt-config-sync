# Making changes

## Style Guide

We follow the following guides:

1.  [K8S Coding Conventions](https://github.com/kubernetes/kubernetes/blob/release-1.1/docs/devel/coding-conventions.md)
1.  [Naming](https://talks.golang.org/2014/names.slide#1)

### Working with the Operator

Nomos is installed via the Nomos Operator, the code for which is in a separate
repository. The
[Nomos Operator Readme](https://team.git.corp.google.com/nomos-team/nomos-operator/+/refs/heads/master/nomos-operator/README.md#working-with-the-nomos-binary-repo)
details how to work with the operator if you need to make changes to both
repositories. The Nomos Operator subdirectory can be found
[here](https://team.git.corp.google.com/nomos-team/nomos-operator/+/refs/heads/master/nomos-operator)

NOTE: The only common case where a modification to this (the Nomos binary)
repository requires a change to the Nomos Operator repository is if the yaml
manifests for Nomos objects are modified. Then, the developer must generate a
new set of manifests for the Operator, as documented in the Operator Readme
[here](https://team.git.corp.google.com/nomos-team/nomos-operator/+/refs/heads/master/nomos-operator/README.md#updating-the-nomos-binary-the-operator-deploys).
Soon after the change to the Nomos binary is merged, the commesurate change to
generate a new set of manifests in the Operator repository should be released as
the latest Release Candidate for the Operator. This can be done by running `make
release` in the `nomos-operator` directory of the `nomos-operator` repository.
This target will create a new Release Canidate, replacing the previous
[operator-latest](https://storage.googleapis.com/nomos-release/operator-latest/nomos-operator.yaml)
release. This is documented in the
[Releasing section of the Operator Readme](https://team.git.corp.google.com/nomos-team/nomos-operator/+/refs/heads/master/nomos-operator/README.md#releasing).
The CI job will likely be broken until this is done, as it will be using an old
build of the operator that does not have the manifest changes you merged.

### Importing versioned packages

Many kubernetes packages are versioned, so we alias their import statements for
clarity:

```go
import (
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
)
```

Although Nomos packages are also versioned, we do not alias them as it is
implied that we are referring to our own types:

```go
import (
    "github.com/google/nomos/pkg/api/configmanagement/v1"
    "github.com/google/nomos/pkg/api/configmanagement/v1alpha1"
)
```

## Commit Message

Convention for commit messages:

```console
Component: Summary of change

Longer description of change addressing as appropriate: why the change is
made,
context if it is part of many changes, description of previous behavior and
newly introduced differences, etc.

Long lines should be wrapped to 80 columns for easier log message viewing in
terminals.

Bug: 123456
```

Some good thoughts on how to write good git commit messages can be found
[here](https://chris.beams.io/posts/git-commit/).

## Go Vendoring (Add a new import)

We use [dep](https://golang.github.io/dep/docs/daily-dep.html) to manage our Go
dependencies.

```console
$ go get -u github.com/golang/dep/cmd/dep
```

If you add an import statement to new a package for the first time:

```console
$ dep ensure
```

To update unconstrained packages:

```console
$ dep ensure -update
```

To update a single package (desired):

```console
$ dep ensure -update -v [package path]
$ ./scripts/fix-dep.sh  # clears out license related changes.
```

## Updating license metadata

Updating dependencies in the `vendor` directory may result in pulling in new
dependencies that need licensing scrutiny.

Before you begin, ensure that your local system has `licenselinter` installed:

```console
$ make install github.com/google/nomos/cmd/licenselinter
```

To check licenses (it is a part of the linter checks, using licenselinter), run:

```console
$ make lint
```

Below is a list of common errors and ways to resolve.

### missing METADATA file, rerun with -generate-meta-file: /some/path/METADATA

From the top level directory of the repo, run:

```console
$ licenselinter --dir=$PWD --generate-meta-file
```

This will generate missing METADATA files for `vendor` dependencies that don't
have them. You can now commit the changes made by the licenselinter.

## Useful Git Tools

git gui is useful for ammending commits

```console
$ sudo apt-get install git-gui
$ git gui
```

gitk is useful for visualizing the commit history

```console
$ sudo apt-get install
$ gitk
```
