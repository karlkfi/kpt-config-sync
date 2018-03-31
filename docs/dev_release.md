# Releasing

In a pristine worktree:

Ensure that your tree does not lag origin.

```shell
git pull --rebase origin master
git status  # should print an empty status
```

Set the release version. Make sure to uphold the semantic versioning rules.

```shell
export RELEASE_VERSION="v1.2.3"
```

Add a version tag.

```shell
git tag -a ${RELEASE_VERSION} -m "Meaningful message"
git push origin ${RELEASE_VERSION}
```

Finally, produce the release. Blessing the release will move the `latest` label
to the one just built.

```shell
make -f Makefile.release release
make -f Makefile.release bless-release
```

To generate a changelog:

```
git log --oneline --decorate=false v0.2.8..v0.2.9
```
