#!/usr/bin/env bats

load ../helpers

function setup(){
  _make create_environment
}

function teardown(){
  _make delete_environment
}

@test  "[${BATS_TEST_FILENAME##*/}] integration-tests (v${CLIENT_VERSION}): create a container with the default logging options" {

  [[ ${CLIENT_VERSION} -ne 5 ]] && skip "this checks the default options which is version 5"

  run docker inspect webapper
  [[ "$status" -eq 0 ]]
  [[ "$(echo ${output} | docker inspect webapper | jq -r '.[0].HostConfig.LogConfig.Config[]' | wc -l)" -eq 1 ]]
  [[ "$(echo ${output} | docker inspect webapper | jq -r '.[0].HostConfig.LogConfig.Config' | jq -r 'keys[]')" == "elasticsearch-url" ]]

}

@test "[${BATS_TEST_FILENAME##*/}] integration-tests (v${CLIENT_VERSION}): check the elasticsearch-version option for different elasticsearch versions" {

  [[ ${CLIENT_VERSION} -eq 5 ]] && skip "elasticsearch version 5 does contain the elasticsearch-version option, because it is the default version"

  run docker inspect webapper
  [[ "$status" -eq 0 ]]
  [[ "$(echo ${output} | docker inspect webapper | jq -r '.[0].HostConfig.LogConfig.Config[]' | wc -l)" -eq 2 ]]
  [[ "$(echo ${output} | docker inspect webapper | jq -r '.[0].HostConfig.LogConfig.Config."elasticsearch-version"')" -eq "$CLIENT_VERSION" ]]

}
