# Releasing

## Setup

In a pristine worktree:

Ensure that your tree does not lag origin.

```console
$ git pull --tags --rebase origin master
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

Find the previous release version.

```console
$ export PREVIOUS_RELEASE=$(git tag --sort=v:refname | tail -n1)
```

Choose a release version. Make sure to uphold the
[semantic versioning rules](http://semver.org). In particular, the new release
version will be major (increment the second number) if there are backwards
incompatible changes, and minor (increment the third number) otherwise.

Backwards-incompatible changes are required to put "BACKWARDS INCOMPATIBLE" in
their commit messages. Check for any such changes:

```console
$ git log $PREVIOUS_RELEASE..HEAD | grep 'BACKWARDS INCOMPATIBLE'
```

Set the release version.

```console
$ export RELEASE_VERSION="v1.2.3"
```

Add a version tag.

```console
$ git tag -a ${RELEASE_VERSION} -m "Weekly release"
$ git push origin ${RELEASE_VERSION}
```

Finally, produce the release candidate:

```console
$ make -f Makefile.release release
```

If the release fails, increment
the patch number for the next release attempt.

Send an email to nomos-team@google.com with subject `Nomos Release
${RELEASE_VERSION}` and body:

```
$ TZ=America/Los_Angeles git log --pretty="format:%C(yellow)%h \
    %C(cyan)%>(12,trunc)%cd %C(green)%<(24,trunc)%aN%C(reset)%s" \
    --date=local ${PREVIOUS_RELEASE}..${RELEASE_VERSION}
```

The artifacts will be
[available here](https://console.cloud.google.com/storage/browser/nomos-release/latest/?project=nomos-release).

## Bless the Release Candidate

We do a manual QA pass before releasing an RC to stable. This includes a sanity check to back up the
e2e tests, as well as any extra testing required for new features.

### Basic manual test

This duplicates the e2e tests, so it's not expected to be comprehensive. This only covers the
possibility that e2e tests are so badly broken that they fail to run at all but still report
passing.

Follow [installation instructions](installation.md), but use the
[latest installer](https://console.cloud.google.com/storage/browser/nomos-release/latest/?project=nomos-release),
which you just created. Follow instructions for [Git config](git_config.md). Use the sample YAML
from those instructions, for the foo-corp repo. You will most likely have memorized these steps, but
try to follow the documentation as best you can. This is our only regular review of the
documentation.

After installation completes, check that the foo-corp namespaces are installed:

```console
$ kubectl get ns
NAME               STATUS    AGE
audit              Active    1m
default            Active    2d
kube-public        Active    2d
kube-system        Active    2d
nomos-system       Active    1m
shipping-dev       Active    1m
shipping-prod      Active    1m
shipping-staging   Active    1m
```

Check that rolebindings are applied:

```console
$ kubectl get ns --as=cheryl@foo-corp.com
(expect same output as above)
```

### Feature QA

Generally, features should be tested by their authors before check-in, and they should be covered
sufficiently by automated tests. If we develop a feature that can't be verified by tests, we will
need a documented process for manual verification before release. For now, automated tests, plus the
above sanity test, are sufficient.

### GCP e2e tests

Run any GCP e2e cases not yet automated:

go/nomos-gcp-e2e-tests

### Blessing

Once manual QA is complete:

```console
$ make -f Makefile.release bless-release
```

The artifacts will be
[available here](https://console.cloud.google.com/storage/browser/nomos-release/stable/?project=nomos-release).
