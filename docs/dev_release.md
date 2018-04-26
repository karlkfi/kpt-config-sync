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

Finally, produce the release. Blessing the release will move the `latest` label
to the one just built.

```console
$ make -f Makefile.release release
$ make -f Makefile.release bless-release
```

To generate a changelog:

```
$ git log --pretty="format:%C(yellow)%h  %C(cyan)%>(15,trunc)%cd %C(green)%<(24,trunc)%aN%C(reset)%s" v0.2.8..v0.2.9
```
