# CHANGELOG

## v0.11.0

### Changelog since v0.10.4

#### Action Required

This is a major change from the previous version, and will require a full
uninstall of the old GKE Policy Management on your cluster. To perform a full
removal of the previous version, run the following commands:

```console
$ kubectl delete ValidatingWebhookConfiguration -l nomos.dev/system=true
$ kubectl delete ns nomos-system
$ kubectl delete customresourcedefinitions -l nomos.dev/system=true
```

In addition, the format of the policy repository has changed. An example of the
new repo format is provided at
[foo corp](https://github.com/frankfarzan/foo-corp-example/tree/0.1.0)

#### Notable changes

*   Moved repository format to Filesystem Standard
    [0.1.0](user/overview.md#filesystem-standard)
*   Added support for [Generic Resource Sync](user/system_config.md#sync)
*   Moved installation to use the [Operator](user/installation.md#installing)
*   Added support for [NamespaceSelectors](user/namespaceselectors.md)
