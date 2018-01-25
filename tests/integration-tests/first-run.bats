#!/usr/bin/env bats

load helpers

function setup(){
  _docker_compose "elasticsearch" "1"
  _app "webapper" "1"
}

function teardown(){
  _make delete_environment
}

@test "send log message to elasticsearch v1" {

  message="${POST_MESSAGE}/${BATS_TEST_NUMBER}"
  _post "$message"

  run _search "$message"
  [[ "$status" -eq 0 ]]

}
