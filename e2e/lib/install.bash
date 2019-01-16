#!/bin/bash

# Helpers for installation

# Creates a public-private ssh key pair.
function install::create_keypair() {
  echo "+++ Creating keypair at: ${HOME}/.ssh"
  mkdir -p "$HOME/.ssh"
  # Skipping confirmation in keygen returns nonzero code even if it was a
  # success.
  (yes | ssh-keygen \
        -t rsa -b 4096 \
        -C "your_email@example.com" \
        -N '' -f "/opt/testing/e2e/id_rsa.nomos") || echo "Key created here."
  ln -s "/opt/testing/e2e/id_rsa.nomos" "${HOME}/.ssh/id_rsa.nomos"
}


function install::nomos_running() {
  if ! kubectl get pods -n nomos-system | grep git-policy-importer | grep Running; then
    echo "Importer not yet running"
    return 1
  fi
  if ! kubectl get pods -n nomos-system | grep syncer | grep Running; then
    echo "Syncer not yet running"
    return 1
  fi
  return 0
}

function install::nomos_uninstalled() {
  if [ "$(kubectl get pods -n nomos-system | wc -l)" -ne 0 ]; then
    echo "Nomos pods not yet uninstalled"
    return 1
  fi
  return 0
}