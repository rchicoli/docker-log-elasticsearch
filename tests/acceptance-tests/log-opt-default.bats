#!/usr/bin/env bats

load ../helpers

function setup(){
  _make create_environment
}

function teardown(){
  _make delete_environment
}

@test "acceptance-tests:$BATS_TEST_NUMBER send log message to elasticsearch v${CLIENT_VERSION}" {

  message="${POST_MESSAGE}/${BATS_TEST_NUMBER}"
  _post "$message"

  run _search "$message"
  [[ "$status" -eq 0 ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerImageName')" ==  "rchicoli/webapper"  ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerName')"      ==  "webapper"           ]]

  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message'     |  _expr  ".*${message}")" -eq 49 ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerID' |  _expr  '[a-z0-9]*')" -eq 12    ]]

  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.source')"  ==  "stderr" ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.partial')" ==  "false"  ]]

}
