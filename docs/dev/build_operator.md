# Building (Operator method)

The instructions for configuring and installing Nomos with an operator are
described in the [Operator Installation Guide](../user/installation.md).
*However*, the `operator-bundle.yaml` uses a released version of Nomos. In order
to deploy your local version run

```$bash
$ make deploy-operator
```

This is instead of downloading and applying the operator bundle (replaces the
steps
[Download Operator Manifest Bundle](../user/installation.md#download-operator-manifest-bundle)
and [Deploy the Operator](../user/installation.md#deploy-the-operator) in the
installation guide).

The `deploy-operator` target assembles and applies a bundle that will deploy
images pushed from your local tree.

If you work in a project shared with other developers and you wish to test with
a build of the operator other than the project-shared version `deploy-operator`
uses, check out the operator source and run `make release-user` there, then run
`make deploy-operator-user` in this repo.
