#!/bin/bash

# Helpers for changing the state of the e2e source-of-truth git
# repository. Prepare the next state to commit using git_add, git_update, and
# git_rm and then commit these changes to the sot repo with git_commit.
#
# Example Usage:
# git_add ${TEST_YAMLS}/new_namespace.yaml acme/rnd/newest-prj/namespace.yaml
# git_update ${TEST_YAMLS}/updated_quota.yaml acme/eng/quota.yaml
# git_rm acme/rnd/new-prj/acme-admin-role.yaml

TEST_REPO=${BATS_TMPDIR}/repo

# Create a new file in the sot directory.
#
# Usage:
# sot_create SOURCE_FILE DEST_LOC
#
# SOURCE_FILE is the location of the source yaml to copy
# DEST_LOC is the path within the sot directory to copy to
#
# Directories that don't exist in the DEST_LOC path will be created.
# If both params are not provided and non-empty, or if the destination
# already exists, the function will fail the containing bats test.
#
function git_add() {
  local src=$1
  local dst=$2

  cd ${TEST_REPO}

  [ ! -z $src ] || (echo "source not provided to git_add" && false)
  [ ! -z $dst ] || (echo "destination not provided to git_add" && false)
  [ ! -f $dst ] || (echo "$dst already exists but should not" && false)

  mkdir -p $(dirname $dst)
  cp $src ./$dst
  git add $dst

  cd -
}

# Update a file in the sot directory.
#
# Usage:
# sot_update SOURCE_FILE DEST_LOC
#
# SOURCE_FILE is the location of the source yaml to copy
# DEST_LOC is the path within the sot directory to copy to
#
# If both params are not provided and non-empty, or if the destination
# does not already exist, the function will fail the containing bats test.
#
function git_update() {
  local src=$1
  local dst=$2

  cd ${TEST_REPO}

  [ ! -z $src ] || (echo "source not provided to git_update" && false)
  [ ! -z $dst ] || (echo "destination not provided to git_update" && false)
  [ -f $dst ] || (echo "$dst does not already exist but should" && false)

  cp $src ./$dst
  git add $dst

  cd -
}

# Delete a file in the sot directory.
#
# Usage:
# sot_delete FILEPATH
#
# FILEPATH is the path within the sot directory to delete
#
# If the param is not provided and non-empty, or if the filepath
# does not already exist, the function will fail the containing bats test.
#
function git_rm() {
  local path=$1

  cd ${TEST_REPO}

  [ ! -z $path ] || (echo "filename not provided to git_rm" && false)
  [ -f $path ] || (echo "$path does not already exist but should" && false)

  git rm $path

  cd -
}

# Commit and push the add, update, and rm operations that are queued
#
# Usage:
# git_commit
#
function git_commit() {
  cd ${TEST_REPO}

  git commit -m "commit for test"
  git push origin master

  cd -
}
