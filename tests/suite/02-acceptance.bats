#!/usr/bin/env bats

load ../helpers

@test "[${BATS_TEST_FILENAME##*/}] suite (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - acceptance tests" {

  for file in $(ls -1 "${TESTS_DIR}/acceptance-tests"); do

    export SKIP="true"
    run bats "${TESTS_DIR}/acceptance-tests/${file}"
    [[ "$status" -eq 0 ]] || _debug "$output"

    export SKIP="false"
    _make delete_environment &>/dev/null

  done

}
