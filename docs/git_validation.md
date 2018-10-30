# Validation

Before committing changes to Git and pushing changes to Kubernetes clusters, it
is critical to validate them first.

`nomosvet` is tool that validates a GKE Policy Management directory by:

1.  Enforcing
    [GKE Policy Management Filesystem Standard](git_overview.md#filesystem-standard).
2.  Validating resources using the Kubernetes API machinery discovery mechanism
    and OpenAPI spec (Similar to `kubectl apply --dry-run`).

To install nomosvet:

```console
$ curl -LO https://storage.googleapis.com/nomos-release/stable/linux_amd64/nomosvet
$ chmod u+x nomosvet
```

You can replace `linux_amd64` in the URL with other supported platforms:

*   `darwin_amd64`
*   `windows_amd64`

The following commands assume that you placed `nomosvet` in a directory
mentioned in your `$PATH` environment variable.

You can manually run nomosvet:

```console
$ nomosvet foo-corp
```

You can also automatically run nomosvet as a git
[pre-commit hook](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks). In
the root of the repo, run:

```console
$ echo "nomosvet foo-corp" > .git/hooks/pre-commit; chmod +x .git/hooks/pre-commit
```

You can also integrate this into your CI/CD setup, e.g. when using GitHub
[required status check](https://help.github.com/articles/about-required-status-checks/).

## Print CRDs

As discussed in [System Overview](system_overview.md), contents of the Git repo
are converted to ClusterPolicy and PolicyNode CRDs during the import process. To
print the generated CRD resources in JSON:

```console
$ nomosvet -print foo-corp
```

This can be handy to preview the diff of a change before it is committed.
