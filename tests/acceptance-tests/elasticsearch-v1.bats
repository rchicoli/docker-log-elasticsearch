#!/usr/bin/env bats

load ../helpers

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
  [[ "$(echo ${output} | _jq 'containerImageName')" ==  "rchicoli/webapper"  ]]
  [[ "$(echo ${output} | _jq 'containerName')"      ==  "webapper"           ]]

  [[ "$(echo ${output} | _jq 'message'     |  _expr  ".*${message}")" -eq 49 ]]
  [[ "$(echo ${output} | _jq 'containerID' |  _expr  '[a-z0-9]*')" -eq 12    ]]

  [[ "$(echo ${output} | _jq 'source')"  ==  "stderr" ]]
  [[ "$(echo ${output} | _jq 'partial')" ==  "false"  ]]

}
