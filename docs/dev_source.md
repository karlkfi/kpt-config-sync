# Making changes

## Style Guide

We follow the following guides:

1.  [K8S Coding
    Conventions](https://github.com/kubernetes/kubernetes/blob/release-1.1/docs/devel/coding-conventions.md)
1.  [Naming](https://talks.golang.org/2014/names.slide#1)

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
