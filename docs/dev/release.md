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

If the release fails, increment the patch number for the next release attempt.

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

We do a manual QA pass before releasing an RC to stable. This includes a sanity
check to back up the e2e tests, as well as any extra testing required for new
features.

### Basic manual test

This duplicates the e2e tests, so it's not expected to be comprehensive. This
only covers the possibility that e2e tests are so badly broken that they fail to
run at all but still report passing.

Follow [installation instructions](../installation.md), but use the
[latest installer](https://console.cloud.google.com/storage/browser/nomos-release/latest/?project=nomos-release),
which you just created. Follow instructions for [Git config](../git_config.md). Use
the sample YAML from those instructions, for the foo-corp repo. You will most
likely have memorized these steps, but try to follow the documentation as best
you can. This is our only regular review of the documentation.

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

Generally, features should be tested by their authors before check-in, and they
should be covered sufficiently by automated tests. If we develop a feature that
can't be verified by tests, we will need a documented process for manual
verification before release. For now, automated tests, plus the above sanity
test, are sufficient.

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

Publicly-accessible docs will be
[available here](https://storage.googleapis.com/nomos-release/stable/nomos-docs.zip)

# Continuous deployment (CD) [EXPERIMENTAL]

This is the release process that promotes an already built release candidate
from the continuous deployment CD pipeline. The deployment process is mostly
automated, but requires a human to bless a release candidate at the very end.

This is what the release process looks like:

- Once a day, the CD pipeline adds a "release candidate" git tag to the
  then-current head revision of the nomos `master` branch.  The tag is of the
  form `v1.2.3-rc.4` which means "4th release candidate for a release `v1.2.3`".

- The CD pipeline runs unit and end-to-end tests based off of the candidate
  release code.

- If the release candidate fails the tests, the CD pipeline stops here.

- If the release candidate passes the tests, it is copied out to `nomos-release`
  project and becomes available as an unblessed release `v1.2.3-rc.4`.

- A nomos release engineer manually runs on their workstation:

```console
make -f Makefile.release bless-autorelease
```

- The above command line launches an interactive prompt that shows the latest
  release candidate ("rc") tags that correspond to last successfully tested release.
  It prints the change log for inspection.

- Release engineer checks the following:

  - Is the previous release tag as expected?  Normally, the previous release tag
    is the previous successful release.

  - Is the release candidate tag as expected?  Normally, the release candidate
	has a patch level one higher than the previous release.  Exceptionally, the
	minor release number is incremented if there are backwards incompatible
	changes.

  - Is the proposed release tag as expected?

- If the release tags proposed by the CD pipeline are as expected, the release
  engineer can press "Enter" to accept defaults.

  - If the release tags proposed by the CD pipeline are not as expected, the
    release engineer may (1) supply other tags or (2) stop the process.

- The blessing process will promote the release candidate `v1.2.3-rc.4` to the
  release `v1.2.3`.

A sample session with `make bless-autorelease` is shown below.

```console
$ make -f Makefile.release bless-autorelease
+++
+++ Bless an automated release from go/nomos-releaser
+++
+++
+++ Previous release tag:                             v0.10.2
+++ Last release candidate tag found in the git repo: v0.10.3-rc.1
+++ Proposed release tag:                             v0.10.3
+++
+++ Proposed changelog is below.
Release candidate to promote (default=v0.10.3-rc.1):
Blessed release version      (default=v0.10.3):
  v0.10.2..v0.10.3-rc.1
36f6fd0f Fri Sep 7 .. Filip Filmar            Makefile.release: adds 'autorelease' target
c79426d4 Fri Sep 7 .. Filip Filmar            buildenv: upgrade buildenv to v0.1.1
567e632f Fri Sep 7 .. Erik Kitson             Log errors from gcloud command without failing.
6eb05e1b Thu Sep 6 .. Brian Thomas Kennedy    Integrate quota flattening with policy importer.
1efc252d Thu Sep 6 .. Brian Thomas Kennedy    Integrate role binding flattening logic with policy importer.
7e28409d Thu Sep 6 .. Brian Thomas Kennedy    Preserve legacy behavior for RoleBinding inheritance.
e15b4b28 Thu Sep 6 .. Erik Kitson             Fix broken links in documentation and add monitor pod.

(...elided...)

Release notes v0.10.2..v0.10.3:

927518c8 Fri Sep 7 .. Filip Filmar            Makefile.release: adds `bless-autorelease`
37d06af5 Fri Sep 7 .. Filip Filmar            Makefile.release: adds continous deployment
d0d160eb Fri Sep 7 .. Phillip Oertel          Update deps.
97f9ecd0 Fri Sep 7 .. Erik Kitson             Fix ignore.bash by ignoring lint-bash's terrible advice.
50224e0b Fri Sep 7 .. Brian Thomas Kennedy    Split makefile rules into several makefiles.
36f6fd0f Fri Sep 7 .. Filip Filmar            Makefile.release: adds 'autorelease' target
c79426d4 Fri Sep 7 .. Filip Filmar            buildenv: upgrade buildenv to v0.1.1
567e632f Fri Sep 7 .. Erik Kitson             Log errors from gcloud command without failing.
6eb05e1b Thu Sep 6 .. Brian Thomas Kennedy    Integrate quota flattening with policy importer.
1efc252d Thu Sep 6 .. Brian Thomas Kennedy    Integrate role binding flattening logic with policy importer.
7e28409d Thu Sep 6 .. Brian Thomas Kennedy    Preserve legacy behavior for RoleBinding inheritance.
e15b4b28 Thu Sep 6 .. Erik Kitson             Fix broken links in documentation and add monitor pod.
```

## Use of release candidates

Release candidates are used in the automated release process to make a clear
distinction between blessed and non-blessed builds.  This was not necessary in
the case of a manual release, since a human would normally figure out the
correct tags to use to produce a new blessed release. This is why we did not
use the release candidate tags and instead just skipped blessing failed release
versions altogether.

Since an automated build doesn't really know what a successful release
candidate looks like without human help, it can not follow the same approach.
A slight change in the release marking conventions was needed to allow
automated builds to proceed without interfering with the manual release
process, which will probably stay around for a while, as initially more mature,
and in the end as an escape hatch.

So, automated builds always produce release candidates and then a human is
needed to promote them to stable.  The `bless-autorelease` target makes an
effort to give reasonable defaults, so in the best case the operator only needs
to press the "Enter" key twice.  Also, the automated release process does not
interfere with the manual one, and the manual releases can continue on
unchanged, for as long as needed.  For the purpose of a manual release (`make
release` and `make bless-release`), the human operator can simply ignore all
tags that have a suffix `-rc` such as v1.2.3-rc.4.

