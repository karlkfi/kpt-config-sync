# Releasing

## Setup

In a pristine worktree:

Ensure that your tree does not lag origin.

```console
$ git pull --rebase origin master
$ git status  # should print an empty status
```

Ensure you're connected to a kubernetes cluster, following
[Setup](setup.md#initial-setup-of-your-cluster-in-gce-one-time). (This will be
used in e2e tests, which run automatically before the release.) You can verify
quickly using

```console
$ kubectl get ns # lists the 3 default namespaces
```

## Create the Release Candidate

Set the release version. Make sure to uphold the
[semantic versioning rules](http://semver.org). If the release fails, increment
the patch number for the next release attempt.

```console
$ export RELEASE_VERSION="v1.2.3"
```

Add a version tag.

```console
$ git tag -a ${RELEASE_VERSION} -m "Meaningful message"
$ git push origin ${RELEASE_VERSION}
```

Finally, produce the release candidate:

```console
$ make -f Makefile.release release
```

Send an email to nomos-team@google.com with subject `Nomos Release
${RELEASE_VERSION}` and body:

```
$ TZ=America/Los_Angeles git log --pretty="format:%C(yellow)%h \
    %C(cyan)%>(12,trunc)%cd %C(green)%<(24,trunc)%aN%C(reset)%s" \
    --date=local v0.3.4..v0.4.0
```

The artifacts will be
[available here](https://console.cloud.google.com/storage/browser/nomos-release/latest/?project=nomos-release).

## Bless the Release Candidate

It is the responsibility of oncall primary and secondary to vet the RC. Wait
until 24 hours to give others a chance to try out the RC as well.

Finally, to bless the RC:

```console
$ make -f Makefile.release bless-release
```

The artifacts will be
[available here](https://console.cloud.google.com/storage/browser/nomos-release/stable/?project=nomos-release).
