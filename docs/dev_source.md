# Making changes

## Style Guide

We follow the following guides:

1.  [K8S Coding
    Conventions](https://github.com/kubernetes/kubernetes/blob/release-1.1/docs/devel/coding-conventions.md)
1.  [Naming](https://talks.golang.org/2014/names.slide#1)

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
