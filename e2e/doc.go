// Package e2e and its subpacakages define go e2e tests for Nomos.
//
// We expect this information to change as we integrate this style of test into
// CI, so check back if things aren't working.
//
// TODO(b/159013140): Make a make target for these.
//
// Running the Tests
//
// $ go test ./e2e/... --e2e
//
// You can use all of the normal `go test` flags.
// The `--e2e` is required or else the e2e tests won't run. This lets you run
// go test ./... to just run unit/integration tests.
//
//
// One-time Setup
//
// Start a local docker registry:
// $ docker run -d --restart=always -p "5000:5000" --name kind-registry registry:2
//
//
// Testing the current state of the code
//
// If you don't do this each time you modify the code, you'll be testing the
// currently-pushed image.
//
// 1) Build the nomos image:
// $ make image-nomos
//
// If successful, it outputs a sha like this:
// > sha256:911ed415b430870a79af98a42b361c1fbcd52c97d1b7d1ee5884f852409d6716
//
// 2) Copy the sha (exclude the "sha256:" prefix), and tag the image
// $ docker tag 911ed415b430870a79af98a42b361c1fbcd52c97d1b7d1ee5884f852409d6716 localhost:5000/nomos:latest
//
// 3) Push the image to the local repository:
// $ docker push localhost:5000/nomos:latest
//
// 4) Run the tests
// $ go test ./e2e/... --e2e
//
//
// Debugging
//
// Use --debug to use the debug mode for tests. In this mode, on failure the
// test does not destroy the kind cluster and delete the temporary directory.
// Instead, it prints out where the temporary directory is and how to connect to
// the kind cluster.
//
// The temporary directory includes:
// 1) All manifests used to install ConfigSync
// 2) The private/public SSH keys to connect to git-server
// 3) The local repository(ies), already configured to talk to git server.
//      Just remember to port-forward to the git-server Pod if you want to read
//      from/write to it.
//
// If you want to stop the test at any time, just use t.FailNow().
package e2e
