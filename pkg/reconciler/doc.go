// Package reconciler declares the reconciler process which is described in
// go/config-sync-multi-repo. This process has four main components:
// git-sync, parser, applier, remediator
//
// git-sync is a third party component that runs in a sidecar container. The
// other three components are all declared in their respective packages.
package reconciler
