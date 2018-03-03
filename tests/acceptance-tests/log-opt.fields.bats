#!/usr/bin/env bats

load ../helpers

function teardown(){
  _make delete_environment
}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - only static fields are logged" {

  export DOCKER_LOG_OPTIONS="${DOCKER_COMPOSE_DIR}/log-opt.static-fields-only.yml"
  _make create_environment

  message="$BATS_TEST_DESCRIPTION"
  _post "$message"

  run _fields "$message"
  [[ "$status" -eq 0 ]]
  [[ "${lines[0]}" == "message" ]]
  [[ "${lines[1]}" == "partial" ]]
  [[ "${lines[2]}" == "source" ]]
  [[ "${lines[3]}" == "timestamp" ]]
  [[ "${#lines[@]}" -eq 4 ]]

}
