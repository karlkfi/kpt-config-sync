# Releasing

The release process promotes an already-built release candidate from the
continuous deployment (CD) pipeline. It is mostly automated but requires a human
to bless a release candidate at the very end.

## Setup

Set up a nomos repo.

Ensure that your remote is named `origin`. `make bless-autorelease` assumes
that.

## Basic manual test

*This duplicates the e2e tests, so it's not expected to be comprehensive. This
only covers the possibility that e2e tests are so badly broken that they fail to
run at all but still report passing. It also tests that our documentation is
accurate (which can't be automated).*

Follow [installation instructions](../user/installation.md), BUT instead of
downloading the `operator-stable` release as instructed, use the latest release at https://storage.cloud.google.com/nomos-release/operator-latest/nomos-operator.yaml

Follow instructions for [Git config](../user/config.md). Use the Nomos YAML
from those instructions, for the foo-corp repo. You will most likely have
memorized these steps, but try to follow the documentation. This is our only
regular review of the documentation.

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

After you have completed release candidate testing, it is time to bless the release.
This will convert an existing release candidate into a version that has its own (non-rc)
version number and will replace the version of the operator bundle at `operator-stable`. For example, blessing v0.2.4-rc.6 will release version v0.2.4 and mark it as the current stable version. The Nomos Operator and Nomos binary codebases are in separate repositories, but the two are currently released together because the Nomos Operator must include the Nomos binaries
in its image (this is a limitation of the operator frameork that will change in the
Q1 2019 timeframe). The two binaries are versioned separately, however.

### Anatomy of a Blessed Release

A blessed release consists of three distinct pieces:

* **`nomos-operator.yaml` manifest** | a yaml bundle that specifies the operator deployment and the roles and role bindings necessary to run it. This file specifies a version of the nomos operator image to use. The current stable (blessed) version of this file is at https://storage.cloud.google.com/nomos-release/operator-stable/nomos-operator.yaml
* **`nomos-operator` image** | the container image of Nomos Operator. The current stable version is at gcr.io/nomos-release/nomos-operator: stable . The operator manifest (above) specifies which version of this image to use.
* **`nomos` image** | the container image of the Nomos binary. The current stable version is at gcr.io/nomos-release/nomos:stable . The manifests that are packaged inside the Nomos Operator image specify which version of this image to use.


### Nomos binary

First, bless the Nomos binary release candidate.

```console
make -f Makefile.release bless-autorelease
```

-   The above command line launches an interactive prompt that shows the latest
    release candidate ("rc") tags that correspond to last successfully tested
    release. It prints the change log for inspection.

-   Release engineer checks the following:

    -   Is the previous release tag as expected? Normally, the previous release
        tag is the previous successful release.

    -   Is the release candidate tag as expected? Normally, the release
        candidate has a patch level one higher than the previous release.
        Exceptionally, the minor release number is incremented if there are
        backwards incompatible changes. Also, be sure that it's the same rc you
        tested manually.

    -   Is the proposed release tag as expected?

-   If the release tags proposed by the CD pipeline are as expected, the release
    engineer can press "Enter" to accept defaults.

    -   If the release tags proposed by the CD pipeline are not as expected, the
        release engineer may (1) supply other tags or (2) stop the process.

-   The blessing process will promote the release candidate `v1.2.3-rc.4` to the
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

-   Send an email to nomos-team@google.com with subject `Nomos binary
    Release ${RELEASE_VERSION}` (where `RELEASE_VERSION` is the tag you chose
    for the release). Copy the above changelog into the body. Copy using `Copy
    as HTML` to retain formatting.

### Nomos Operator

Switch to the `nomos-operator` repository. Before blessing, take note of whether the new release should have be a new patch, minor, or major version. By default, the blessing target increments the patch set, so to bless a new version with just a patch increment run:

```console
make bless-release
```

If you would instead like to increment the minor version, run:
```console
MINOR=true make bless-release
```

See the comments on this target in the [Makefile](https://gke-internal.git.corp.google.com/cluster-lifecycle/cluster-operators/+/master/nomos-operator/Makefile#186) for other options available for this target.

The `bless-release` target is interactive; please ensure the version name proposed by the target is what you expect before continuing.

## Check the build artifacts (optional)

In case you're interested:

The artifacts will be
[available here](https://console.cloud.google.com/storage/browser/nomos-release/stable/?project=nomos-release).

Publicly-accessible docs will be
[available here](https://storage.googleapis.com/nomos-release/stable/nomos-docs.zip)

## Appendix: what the CD pipeline does

This is what the release process looks like:

-   Once a day, the CD pipeline adds a "release candidate" git tag to the
    then-current head revision of the nomos `master` branch. The tag is of the
    form `v1.2.3-rc.4` which means "4th release candidate for a release
    `v1.2.3`".

-   The CD pipeline runs unit and end-to-end tests based off of the candidate
    release code.

-   If the release candidate fails the tests, the CD pipeline stops here.

-   If the release candidate passes the tests, it is copied out to
    `nomos-release` project and becomes available as an unblessed release
    `v1.2.3-rc.4`.

-   When the release engineer runs `make bless-autorelease`, it looks for the
    latest tag.
