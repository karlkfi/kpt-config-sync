#!/bin/bash
wdiff -3 kube-apiserver.manifest.orig.m4 kube-apiserver.manifest.patched.m4 > manifest.diff
