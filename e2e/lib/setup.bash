#!/bin/bash

set -e

TEST_REPO_DIR=${BATS_TMPDIR}

# Total count of namespaces in acme
ACME_NAMESPACES=(
  analytics
  backend
  frontend
  new-prj
  newer-prj
)

# System namespaces (kubernetes + nomos)
SYS_NAMESPACES=(
  default
  kube-public
  kube-system
  nomos-system
  nomos-system-test
)

# Make "prefix" short, so that project and folder names don't go over name
# length limit.  This is not ironclad, but should work for most things. The
# chosen length is completely arbitrary.
readonly prefix="${USER:0:11}"

# Make available the general settings for running an end-to-end GCP test.
# Since we are not allowed to create ephemeral projects in the e2e tests by
# Elysium policy, some state must be reused across runs.  We set up that state
# here so it is available to the test cases, and we create one test project per
# different $USER.
export GCP_ORG_ID="495131404417" # nomos-e2e.joonix.net
export GCP_TEST_NAMESPACE="${prefix}-nomos-e2e"
export GCP_PROJECT_A="${GCP_TEST_NAMESPACE}-sf"
export GCP_PROJECT_B="${GCP_TEST_NAMESPACE}"
export GCP_FOLDER="$GCP_TEST_NAMESPACE-folder"

setup::gcp::delete_namespace() {
  local namespace="$1"
  local project_id="${namespace}"
  echo "setup::gcp::delete_namespace namespace=${namespace}"
  local project_num
  project_num=$(gcloud projects describe "${project_id}" --format="get(projectNumber)")
  echo "setup::gcp::delete_namespace project_num=${project_num}"

  # || true to ignore scenario where the namespace is not found.
  gcloud --quiet alpha container policy namespaces delete \
          --project="${project_id}" \
          "projects/${project_num}/namespaces/${namespace}" || true
  namespace::check_not_found -t 120 "${namespace}"
  echo "setup::gcp::delete_namespace exit"
}

# Creates a project with the given id if it doesn't already exist
setup::gcp::create_project() {
  local project="$1"
  echo "setup::gcp::create_project=${project}"

  if ! gcloud projects describe "${project}" &> /dev/null ; then
    echo gcloud exit code: $?
    # If an e2e test project does not exist for this user, create it and
    # do what is necessary to make it a functional e2e project.
    gcloud projects create \
        --organization="${GCP_ORG_ID}" \
        "${project}"
    echo "IF RUNNING FOR THE FIRST TIME THIS WILL FAIL. See b/111757245"

    echo "setup::gcp enable services"
    gcloud --quiet services enable \
        kubernetespolicy.googleapis.com --project="${project}"
  fi
}

# Sets the FOLDER_ID variable to the ID of the folder with the given display
# name. Creates the folder if needed.
setup::gcp::create_folder() {
  local folder_display_name="$1"

  echo "setup::gcp::set_or_create_folder=${folder_display_name}"

  # This returns the folder number if exists
  run gcloud alpha resource-manager folders list --organization "${GCP_ORG_ID}" \
    --filter=display_name:"${folder_display_name}" --format="value(name)"

  # shellcheck disable=SC2154
  if [[ -z $output ]]; then
    # This will return folder name "folders/foldernumber"
    run gcloud alpha resource-manager folders create --display-name="${folder_display_name}" \
      --organization "${GCP_ORG_ID}" --format="value(name)"
    # shellcheck disable=SC2154
    [ "$status" -eq 0 ]
    FOLDER_ID=$(echo "$output" | awk '/^folders/' | cut -d / -f 2) # extract foldernumber
  else
    FOLDER_ID=$output
  fi
  export FOLDER_ID
  echo "Folder id=${FOLDER_ID} found"

}

# GCP Initialize will set the gcloud auth to use the test_runner_creds
# And ensure you have a clean hierarchy of a project, a folder, and another
# project under the folder with no namespaces and clean rolebindings.
#
# GCP_ORG
# |_ GCP_FOLDER
# |  |_ GCP_PROJECT_A
# |_ GCP_PROJECT_B
#
setup::gcp::initialize() {
  echo "setup::gcp::initialize"
  GCLOUD_CONTEXT="$(gcloud config get-value account)"
  gcloud auth activate-service-account test-runner@nomos-e2e-test1.iam.gserviceaccount.com \
          --key-file="$HOME/test_runner_client_key.json"

  setup::gcp::create_project "${GCP_PROJECT_A}"
  setup::gcp::create_project "${GCP_PROJECT_B}"
  setup::gcp::create_folder "${GCP_FOLDER}"

  gcloud alpha projects move "${GCP_PROJECT_A}" --folder="${FOLDER_ID}"
  gcloud alpha projects move "${GCP_PROJECT_B}" \
      --organization="${GCP_ORG_ID}"

  setup::gcp::delete_namespace "${GCP_PROJECT_A}"
  setup::gcp::delete_namespace "${GCP_PROJECT_B}"

  gcloud projects remove-iam-policy-binding "${GCP_PROJECT_A}" \
      --member=user:bob@nomos-e2e.joonix.net --role=roles/container.viewer || true
  gcloud projects remove-iam-policy-binding "${GCP_PROJECT_B}" \
      --member=user:bob@nomos-e2e.joonix.net --role=roles/container.viewer || true
  gcloud alpha resource-manager folders remove-iam-policy-binding "${FOLDER_ID}" \
      --member=user:bob@nomos-e2e.joonix.net --role=roles/container.viewer || true

  echo "setup::gcp::initialize exit"
}

gcp::teardown() {
  echo "gcp::teardown"
  gcloud config set account "${GCLOUD_CONTEXT}"
  echo "gcp::teardown exit"
}

# Git-specific repository initialization.
setup::git::initialize() {
  # Reset git repo to initial state.
  CWD="$(pwd)"
  echo "Setting up local git repo"
  rm -rf "${TEST_REPO_DIR}/repo"
  mkdir -p "${TEST_REPO_DIR}/repo"
  cd "${TEST_REPO_DIR}/repo"

  git init
  git remote add origin ssh://git@localhost:2222/git-server/repos/sot.git
  git fetch
  git config user.name "Testing Nome"
  git config user.email testing_nome@example.com
  mkdir acme
  touch acme/README.md
  git add acme/README.md
  git commit -a -m "initial commit"

  cp -r /opt/testing/e2e/sot/acme ./
  git add -A
  git commit -m "setUp commit"
  git push origin master:master -f
  cd "$CWD"

  setup::wait_for_namespaces

  # Wait for syncer to update each object type.
  wait::for kubectl get rolebindings -n backend
  wait::for kubectl get roles -n new-prj
  wait::for kubectl get quota -n backend
  # We delete bob-rolebinding in one test case, make sure it's restored.
  wait::for kubectl get rolebindings bob-rolebinding -n backend
  wait::for kubectl get clusterrole acme-admin

  if type local_setup &> /dev/null; then
    echo "Running local_setup"
    local_setup
  fi
}

setup() {
  E2E_TEST_FILTER="${E2E_TEST_FILTER:-}"
  if [[ "${E2E_TEST_FILTER}" != "" ]]; then
    local cur_test=""
    for func in "${FUNCNAME[@]}"; do
      if echo "$func" | grep "^test_" &> /dev/null; then
        cur_test="${func}"
      fi
    done
    if ! echo "${cur_test}" | grep "${E2E_TEST_FILTER}" &> /dev/null; then
      skip
    fi
  fi

  # Delete testdata that might exist.
  kubectl delete ns -l "nomos.dev/testdata=true" &> /dev/null || true

  case "${IMPORTER}" in
    git)
      setup::git::initialize
	  ;;
    gcp)
      setup::gcp::initialize
      ;;
    *)
      echo "Invalid importer: ${IMPORTER}"
  esac
  echo "--- SETUP COMPLETE ---------------------------------------------------"
}

# Previous tests can create / delete namespaces. This will wait for the
# namespaces to finish terminating and the state to get restored to base acme.
function setup::wait_for_namespaces() {
  debug::log -n "Waiting for namespaces to stabilize"
  local start
  start=$(date +%s)
  local deadline
  deadline=$(( $(date +%s) + 30 ))
  while (( "$(date +%s)" < deadline )); do
    if setup::check_stable; then
      debug::log "Namespaces stabilized in $(( $(date +%s) - start )) seconds"
      return 0
    fi
    sleep 0.1
  done
  debug::log "Namespaces failed to stabilize to acme defaults, got:"
  debug::log "$(kubectl get ns)"
  return 1
}

function setup::check_stable() {
  debug::log "checking for stable"
  local ns_states
  ns_states="$(
    kubectl get ns -ojsonpath="{.items[*].status.phase}" \
    | tr ' ' '\n' \
    | sort \
    | uniq -c
  )"
  if echo "${ns_states}" | grep "Terminating" &> /dev/null; then
    local count
    count=$(echo "${ns_states}" | grep "Terminating" | sed -e 's/^ *//' -e 's/T.*//')
    debug::log "Waiting for $count namespaces to finalize"
    return 1
  fi

  local ns_count
  ns_count=$(resource::count -r ns -l nomos.dev/namespace-management)
  if (( ns_count != ${#ACME_NAMESPACES[@]} )); then
    debug::log "count mismatch $ns_count != ${#ACME_NAMESPACES[@]}"
    return 1
  fi

  echo "Checking namespaces for active state"
  for ns in "${ACME_NAMESPACES[@]}" "${SYS_NAMESPACES[@]}"; do
    if ! kubectl get ns "${ns}" -oyaml | grep "phase: Active" &> /dev/null; then
      debug::log "namespace ${ns} not active yet"
      return 1
    fi
  done
}

teardown() {
  echo "--- BEGINNING TEARDOWN -----------------------------------------------"
  if type local_teardown &> /dev/null; then
    echo "Running local_teardown"
    local_teardown
  fi
  if [[ "$IMPORTER" == "gcp" ]]; then
    gcp::teardown
  fi
}

