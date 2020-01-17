# Release Overview

Anthos Config Management is an on-prem, enterprise security software. As such, we
want to make sure that the release process is well-documented and easy to
follow. At a high level, the oncall performs the following steps to do a
release:

1.  Announce the intent to bless "latest"
2.  Baking in the "latest"
3.  The Blessing
4.  Announce the blessed

Each step is discussed below.

# 1. Announce the intent to bless "latest"

Send an email to nomos-team@google.com with subject `Nomos Release Candidate
2019/XX/YY`.

# 2. Baking in the "latest"

Allow at least 24 hours for this step. During this period, the team has a chance
to try out "latest". Note that currently "latest" is a floating tag and may
refer to different binaries depending on when it's pulled (This will be improved
in the future).

Depending on the features being released, importance of the launch, and state of
the docs, we may want to set up a fishfooding session. However, fishfooding is
not a substition for automated testing. Features (including related docs) should
be tested by their authors before check-in, and they should be covered
sufficiently by automated tests.

### Fishfood Candidate

Fishfood should be done off the current latest release. Usually this is the
release candidate that was produced last night, so no manual action is required
to create this candidate.

If it is desired to manually create a latest release, run `make release` in the
`nomos-operator` subdirectory of the nomos-operator repo. See the section on
[Working with the Operator](source.md#working-with-the-operator) for more
information.

# 3. The Blessing

Blessing process will release an RC to the end user.

Blessing will convert an existing release candidate into a version that has its
own (non-rc) version number and will replace the version of the operator bundle
at `operator-stable`. For example, blessing v0.2.4-rc.6 will release version
v0.2.4 and mark it as the current stable version. The Nomos Operator and Nomos
binary codebases are in separate repositories, but the two are currently
released together because the Nomos Operator must include the Nomos binaries in
its image (this is a limitation of the operator frameork that will change in the
Q1 2019 timeframe). The two binaries are versioned separately, however.

### FYI: How CD Pipeline works

These are the steps performed by the CD pipeline running on Prow:

1.  Once a day, the CD pipeline adds a "release candidate" git tag to the
    then-current head revision of the nomos `master` branch. The tag is of the
    form `v1.2.3-rc.4` which means "4th release candidate for a release
    `v1.2.3`".
1.  The CD pipeline runs unit and end-to-end tests based off of the candidate
    release code.
1.  If the release candidate fails the tests, the CD pipeline stops here.
1.  If the release candidate passes the tests, it is copied out to
    `nomos-release` project and becomes available as an unblessed release
    `v1.2.3-rc.4`.
1.  When the release engineer runs `make bless-autorelease`, it looks for the
    latest tag.

### Anatomy of a Blessed Release

A blessed release consists of three distinct pieces:

*   **`config-management-operator.yaml` manifest** | a yaml bundle that specifies the
    operator deployment and the roles and role bindings necessary to run it.
    This file specifies a version of the nomos operator image to use. The
    current stable (blessed) version of this file is at
    https://storage.cloud.google.com/nomos-release/operator-stable/config-management-operator.yaml
*   **`nomos-operator` image** | the container image of Nomos Operator. The
    current stable version is at gcr.io/nomos-release/nomos-operator:stable .
    The operator manifest (above) specifies which version of this image to use.
*   **`nomos` image** | the container image of the Nomos binary. The current
    stable version is at gcr.io/nomos-release/nomos:stable . The manifests that
    are packaged inside the Nomos Operator image specify which version of this
    image to use.

### Verify the Release Candidate

Before blessing, visit [go/nomos-verify-release](http://go/nomos-verify-release)
and verify that the most recent test run (at 11PM last night) ran and ran
successfully. If it did not, the corresponding release should not be blessed.

### Nomos binary

First, bless the Nomos binary release candidate.

```console
make -f Makefile.release bless-release
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

A sample session with `make bless-release` is shown below.

```console
$ make -f Makefile.release bless-release
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

### Nomos Operator

#### Prerequisites

You will need the following to properly bless the release:

*   dep 0.5.0.
*   a checked out
    [`nomos-operator` repository](https://g3doc.corp.google.com/company/teams/nomos-team/dev_guide.md#operator)

Here's how to check:

```console
$ dep version
dep:
 version     : v0.5.0
 build date  : 2018-07-26
 git hash    : 224a564
 go version  : go1.10.3
 go compiler : gc
 platform    : linux/amd64
 features    : ImportDuringSolve=false
```

Switch to the `nomos-operator`
[repository](https://g3doc.corp.google.com/company/teams/nomos-team/dev_guide.md#operator).
Before blessing, take note of whether the new release should have be a new
patch, minor, or major version. By default, the blessing target increments the
patch set, so to bless a new version with just a patch increment run:

```console
make bless-release
```

If you would instead like to increment the minor version, run: `console
MINOR=true make bless-release`

See the comments on this target in the
[Makefile](https://team.git.corp.google.com/nomos-team/nomos-operator/+/refs/heads/master/nomos-operator/Makefile#186)
for other options available for this target.

The `bless-release` target is interactive; please ensure the version name
proposed by the target is what you expect before continuing.

#### NOTE: In case your release attempt fails

Due to an as of yet unknown cause, running `dep ensure -v` which is part of the
release process fails very often. This is a collection of steps that *may* help
you stop the bleeding while we figure out what actually is going on.

A confounding factor is that the errors you may see seem nondeterministic.

If you encounter the error `fatal: failed to unpack tree object` you will need
to clean up the repository manually as follows. Once you clean up the repository
you can re-attempt `make bless-release`.

```console
$ git add -A
$ git reset --hard HEAD  # this and above command will clean up the git index
$ rm -fr ../.vendor-new  # this removes a dep artifact
```

You might need to remove also the directory `$GOPATH/pkg/dep/sources` which
contains the packages cached by dep.

### Check the build artifacts (optional)

In case you're interested:

The artifacts will be
[available here](https://console.cloud.google.com/storage/browser/nomos-release/stable/?project=nomos-release).

# 4. Announce the blessed

Reply to the email sent in step #1, announcing the RC was blessed with the body:

```
Nomos binary blessed: ${RELEASE_VERSION}

${Changelong in HTML format}
```
