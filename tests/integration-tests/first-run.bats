#!/usr/bin/env bats

load ../helpers

function setup(){
  _make create_environment
}

function teardown(){
  _make delete_environment
}

@test "integration-tests: create a container with elasticsearch logging driver v${CLIENT_VERSION}" {

  run docker inspect webapper
  [[ "$status" -eq 0 ]]
  [[ "$(echo ${output} | docker inspect webapper | jq -r '.[0].HostConfig.LogConfig.Config."elasticsearch-version"')" -eq $CLIENT_VERSION ]]

}
