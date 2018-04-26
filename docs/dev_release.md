# Releasing

In a pristine worktree:

Ensure that your tree does not lag origin.

```console
$ git pull --rebase origin master
$ git status  # should print an empty status
```

Set the release version. Make sure to uphold the semantic versioning rules.

```console
$ export RELEASE_VERSION="v1.2.3"
```

Add a version tag.

```console
$ git tag -a ${RELEASE_VERSION} -m "Meaningful message"
$ git push origin ${RELEASE_VERSION}
```

Finally, produce the release, which will move the `latest` label
to the one just built and update the latest Nomos docs.

```console
$ make -f Makefile.release release
```
**Important**: If you are confident that this particular
release candidate is functional and stable, then proceed to
mark it as stable. This will move the `stable` tag to the
version just built and update stable
Nomos docs.

```console
$ make -f Makefile.release bless-release
```

To generate a changelog:

```
$ git log --pretty="format:%C(yellow)%h  %C(cyan)%>(15,trunc)%cd %C(green)%<(24,trunc)%aN%C(reset)%s" v0.2.8..v0.2.9
```
