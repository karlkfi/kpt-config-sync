#!/bin/bash

load "../lib/git"
load "../lib/wait"
load "../lib/debug"

function stress::create_many_namespaces() {
  local num_namespaces
  num_namespaces="${1:-10}"

  local ns_file
  ns_file="$(mktemp)"
  for index in $(seq -w 1 "${num_namespaces}"); do
    cat <<EOF > "${ns_file}"
kind: Namespace
apiVersion: v1
metadata:
  name: ns-${index}
EOF
    git::add "${ns_file}" "acme/namespaces/ns-${index}/namespace.yaml"

  done

  debug::log "Commit all the namespaces"
  git::commit -m "Commit all the namespaces"

  debug::log "Check that namespaces have been created"
  for index in $(seq -w 1 "${num_namespaces}"); do
    # Namespaces are not created in order: this may fail if the earlier
    # namespaces are created very late in the process.
    wait::for -t 1800 -- kubectl get ns "ns-${index}"
    debug::log "Namespace ns-${index} exists"
  done

  rm "${ns_file}"
}

function stress::create_many_resources()  {
  local num_resources
  num_resources="${1:-10}"
  local ns
  ns="test-namespace"

  debug::log "Create namespace ${ns} for tests"
  local ns_file
  ns_file="$(mktemp)"
  cat <<EOF > "${ns_file}"
kind: Namespace
apiVersion: v1
metadata:
  name: ${ns}
EOF
  git::add "${ns_file}" "acme/namespaces/${ns}/namespace.yaml"

  debug::log "Create ${num_resources} resources in ${ns}"
  local resource_file
  resource_file="$(mktemp)"
  for index in $(seq -w 1 "${num_resources}"); do
    cat <<EOF > "${resource_file}"
kind: Service
apiVersion: v1
metadata:
  name: service-${index}
spec:
  sessionAffinity: ClientIP
  selector:
    app: some-service
  ports:
  - name: http
    protocol: TCP
    port: 80
    targetPort: 2${index}
EOF
    git::add "${resource_file}" \
      "acme/namespaces/${ns}/service-${index}.yaml"
  done

  git::commit -a -m "Create many resources in ${ns}"

  debug::log "Quick check that the namespace is there"
  wait::for -t 300 -- kubectl get namespace "${ns}"

  debug::log "Wait for services to appear in the cluster"
  for index in $(seq -w 1 "${num_resources}"); do
    wait::for -t 1800 -- kubectl get service \
      "service-${index}" --namespace="${ns}"
    debug::log "Service service-${index} was created"
  done

  rm -f "${ns_file}" "${resource_file}"
}

function stress::create_many_commits()  {
  local num_resources
  num_resources="${1:-10}"
  local sleep_pause
  sleep_pause="${2:-1}"

  local ns
  ns="test-commit-rate-namespace"

  debug::log "Create namespace ${ns} for tests"
  local ns_file
  ns_file="$(mktemp)"
  cat <<EOF > "${ns_file}"
kind: Namespace
apiVersion: v1
metadata:
  name: ${ns}
EOF
  git::add "${ns_file}" "acme/namespaces/${ns}/namespace.yaml"

  debug::log "Create ${num_resources} resources in ${ns}"
  git::commit -a -m "Create namespace ${ns}"

  local resource_file
  resource_file="$(mktemp)"
  for index in $(seq -w 1 "${num_resources}"); do
    cat <<EOF > "${resource_file}"
kind: Service
apiVersion: v1
metadata:
  name: service-${index}
spec:
  sessionAffinity: ClientIP
  selector:
    app: some-service
  ports:
  - name: http
    protocol: TCP
    port: 80
    targetPort: 2${index}
EOF
    git::add "${resource_file}" \
      "acme/namespaces/${ns}/service-${index}.yaml"
    git::commit -m "Create service-${index}"

    sleep "${sleep_pause}"
  done

  debug::log "Quick check that the namespace is there"
  wait::for -t 300 -- kubectl get namespace "${ns}"

  debug::log "Wait for services to appear in the cluster"
  for index in $(seq -w 1 "${num_resources}"); do
    wait::for -t 1800 -- kubectl get service \
      "service-${index}" --namespace="${ns}"
    debug::log "Service service-${index} was created"
  done

  rm -f "${ns_file}" "${resource_file}"
}
