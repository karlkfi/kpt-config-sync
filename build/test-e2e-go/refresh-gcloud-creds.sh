#!/bin/bash

# The script running in background that refreshes the gcloud credentials every 50 minutes.

set -e

while true; do
  # Delete the lines containing `expiry: ` so that
  # `gcloud config config-helper --force-auth-refresh` can force-update the `access_token`.
  # If expiry is defined, it only updates `access_token` when it expires even if `--force-auth-refresh` is provided.
  sed -i "/expiry: /d" "${HOME}/.kube/config"

  gcloud --quiet config config-helper --force-auth-refresh >/dev/null

  # the gcloud credentials expire every 60 minutes, so force a refresh every 50 minutes.
  sleep 3000
done
