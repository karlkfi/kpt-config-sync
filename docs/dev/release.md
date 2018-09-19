# Releasing

The release process promotes an already-built release candidate
from the continuous deployment (CD) pipeline. It is mostly
automated but requires a human to bless a release candidate at the very end.

## Setup

Set up a nomos repo.

Ensure that your remote is named `origin`. `make bless-autorelease` assumes that.

## Basic manual test

*This duplicates the e2e tests, so it's not expected to be comprehensive. This
only covers the possibility that e2e tests are so badly broken that they fail to
run at all but still report passing. It also tests that our documentation is
accurate (which can't be automated).*

Follow [installation instructions](../installation.md), but instead of `stable`, pick a fixed
release from
[GCS nomos releases](https://console.cloud.google.com/storage/browser/nomos-release?project=nomos-release).
Choose the highest versioned release. It must be a release candidate (rc), such as `v0.10.3-rc.39`.

Follow instructions for [Git config](../git_config.md). Use
the sample YAML from those instructions, for the foo-corp repo. You will most
likely have memorized these steps, but try to follow the documentation. This is our
only regular review of the documentation.

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

## Blessing

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
	changes. Also, be sure that it's the same rc you tested manually.

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

- Final step: send an email to nomos-team@google.com with subject `Nomos Release
${RELEASE_VERSION}` (where `RELEASE_VERSION` is the tag you chose for the release). Copy the above
 changelog into the body. Copy using `Copy as HTML` to retain formatting.

## Check the build artifacts (optional)

In case you're interested:

The artifacts will be
[available here](https://console.cloud.google.com/storage/browser/nomos-release/stable/?project=nomos-release).

Publicly-accessible docs will be
[available here](https://storage.googleapis.com/nomos-release/stable/nomos-docs.zip)

## Appendix: what the CD pipeline does

This is what the release process looks like:

- Once a day, the CD pipeline adds a "release candidate" git tag to the
  then-current head revision of the nomos `master` branch.  The tag is of the
  form `v1.2.3-rc.4` which means "4th release candidate for a release `v1.2.3`".

- The CD pipeline runs unit and end-to-end tests based off of the candidate
  release code.

- If the release candidate fails the tests, the CD pipeline stops here.

- If the release candidate passes the tests, it is copied out to `nomos-release`
  project and becomes available as an unblessed release `v1.2.3-rc.4`.

- When the release engineer runs `make bless-autorelease`, it looks for the latest tag.

