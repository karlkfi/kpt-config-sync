# CHANGELOG

## v0.11.0

### Changelog since v0.10.4

#### Action Required

This is a major change from the previous version, and will require installation
on a clean cluster. An earlier installation of GKE Policy Management will
interfere with an installation of v0.11.0.

In addition, the format of the policy repository has changed. An example of the
new repo format is provided at
[foo corp](https://github.com/frankfarzan/foo-corp-example/tree/0.1.0)

#### Notable changes

*   Moved repository format to Filesystem Standard
    [0.1.0](user/overview.md#filesystem-standard)
*   Added support for [Generic Resource Sync](user/system_config.md#sync)
*   Moved installation to use the [Operator](user/installation.md#installing)
*   Added support for [NamespaceSelectors](user/namespaceselectors.md)
