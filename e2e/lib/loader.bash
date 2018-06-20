

# BATS_TEST_DIRNAME changes based on the path of the .bats test file, so loading
# a lot of libs is a bit troublesome unless you have an absolute path.
LIB_DIR_PATH=/opt/testing/e2e/lib

# This function loads all libs in e2e/lib so we don't have to worry about tests
# failing becuase they don't load a lib.
function libs::load_all() {
  local libname
  for libname in $(ls ${LIB_DIR_PATH}/*.bash); do
    if [[ "$(basename $libname)" == "loader.bash" ]]; then
      continue
    fi
    load ${libname}
  done
}

libs::load_all
