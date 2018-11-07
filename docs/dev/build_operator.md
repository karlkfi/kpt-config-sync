# Building (Operator method)

The instructions for configuring and installing Nomos with an operator are
described in the [Operator Installation Guide](../installation_operator.md).
*However*, the `operator-bundle.yaml` uses a released version of Nomos. In order
to deploy your local version run

```$bash
$ make deploy-operator
```

This is instead of downloading and applying the operator bundle (replaces the
steps
[Download Operator Manifest Bundle](../installation_operator.md#download-operator-manifest-bundle)
and [Deploy the Operator](../installation_operator.md#deploy-the-operator) in
the installation guide).

The `deploy-operator` target assembles and applies a bundle that will deploy
images pushed from your local tree.
