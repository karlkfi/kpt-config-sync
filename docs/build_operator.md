# Building (Operator method)

The instructions for configuring and installing Nomos with an operator are
described in the [Operator Installation Guide](../user/installation.md).
*However*, `config-management-operator.yaml` uses a released build of Nomos. In order to
deploy your local tree with the latest operator release candidate run

```$bash
$ make deploy-operator
```

And add the following to `nomos.yaml`:

```$bash
spec:
  channel: dev
```

This is instead of downloading and applying the operator bundle (replaces the
steps
[Download Operator Manifest Bundle](../user/installation.md#download-operator-manifest-bundle)
and [Deploy the Operator](../user/installation.md#deploy-the-operator) in the
installation guide).

If wish to test with your own dev build of the operator, check out the operator
repository, cd into the `nomos-operator` directory, then run

```$bash
make release-user
```

then change back to this directory and run

```$bash
make deploy-operator-user
```

This will use the operator build in the `nomos-operator` tree where you last ran
`release-user`. Note that if you have never run `release-user` this will be
non-existent.

### Troubleshooting

*Nomos Deployments are stuck in ImagePullBackoff*

If you did not add `channel: dev` to the `nomos.yaml` you applied, kubernetes
will attempt to retrieve images from the incorrect location, resulting in
ImagePullBackoff. To fix this, add `channel: dev` to your `nomos.yaml` and
re-apply.
