#!/usr/bin/env bats

load ../helpers

function setup(){
  _make create_environment
}

function teardown(){
  _make delete_environment
}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - restarting a container" {

  run docker restart webapper
  [[ "$status" -eq 0 ]] || _debug "$output"

  run docker restart webapper
  [[ "$status" -eq 0 ]] || _debug "$output"

  run docker restart webapper
  [[ "$status" -eq 0 ]] || _debug "$output"

}