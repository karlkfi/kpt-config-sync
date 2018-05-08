# Releasing

## Setup

In a pristine worktree:

Ensure that your tree does not lag origin.

```console
$ git pull --rebase origin master
$ git status  # should print an empty status
```

Ensure you're connected to a kubernetes cluster, following
[Setup](dev_setup.md#initial-setup-of-your-cluster-in-gce-one-time). (This will
be used in e2e tests, which run automatically before the release.) You can verify
quickly using

```console
$ kubectl get ns # lists the 3 default namespaces
```

## Release

Set the release version. Make sure to uphold the [semantic versioning
rules](http://semver.org).
If the release fails, increment the patch number for the next release attempt.

```console
$ export RELEASE_VERSION="v1.2.3"
```

Add a version tag.

```console
$ git tag -a ${RELEASE_VERSION} -m "Meaningful message"
$ git push origin ${RELEASE_VERSION}
```

Finally, produce the release, which will move the `latest` label to the one just
built and update the latest Nomos docs.

```console
$ make -f Makefile.release release
```

**Important**: If you are confident that this particular release candidate is
functional and stable, then proceed to mark it as stable. This will move the
`stable` tag to the version just built and update stable Nomos docs.

```console
$ make -f Makefile.release bless-release
```

To generate a changelog:

```
$ TZ=America/Los_Angeles git log --pretty="format:%C(yellow)%h \
    %C(cyan)%>(12,trunc)%cd %C(green)%<(24,trunc)%aN%C(reset)%s" \
    --date=local v0.3.4..v0.4.0
```

## Verify (optional)

If the above commands succeeded (that is, `echo $?` prints 0), the release process
was successful. The output is a new version of our container in gcr. Look for the
new version
[here](https://pantheon.corp.google.com/gcr/images/nomos-release/GLOBAL/installer?project=nomos-release&organizationId=433637338589&gcrImageListsize=50).
