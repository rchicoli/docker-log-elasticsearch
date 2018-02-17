#!/usr/bin/env bats

load ../helpers

function setup(){
  _make create_environment
}

function teardown(){
  _make delete_environment
}

@test "acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - all default fields" {

  message="$BATS_TEST_DESCRIPTION"
  _post "$message"

  run _fields "$message"
  [[ "$status" -eq 0 ]]
  [[ "${lines[0]}" == "containerCreated" ]]
  [[ "${lines[1]}" == "containerID" ]]
  [[ "${lines[2]}" == "containerImageName" ]]
  [[ "${lines[3]}" == "containerName" ]]
  [[ "${lines[4]}" == "message" ]]
  [[ "${lines[5]}" == "partial" ]]
  [[ "${lines[6]}" == "source" ]]
  [[ "${lines[7]}" == "timestamp" ]]
  [[ ${#lines[@]} -eq 8 ]]

}

@test "acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - log messages to elasticsearch with default options" {

  [[ ${CLIENT_VERSION} -eq 1 ]] && skip "elasticsearch version ${CLIENT_VERSION} does not support unicode chars"

  message="$BATS_TEST_DESCRIPTION"
  _post "$message"

  run _search "$message"
  [[ "$status" -eq 0 ]]

  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerCreated' | egrep '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]+Z$')" ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerID'      | egrep '^[a-z0-9]{12}$')" ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerImageName')" == "rchicoli/webapper" ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerName')"      == "webapper"          ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')"            == *"$message"*        ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.source')"             == "stdout"            ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.partial')"            == "false"             ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.timestamp' | egrep '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]+Z$')" ]]
  [[ $(echo ${output} | jq -r '.hits.hits[0]._source[]' | wc -l) -eq 8 ]]

}

@test "acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - log unicode chars" {

  message="${BATS_TEST_NUMBER}:héllö-yöü ❤ ☀ ☆ ☂ ☻ ♞ ☯ ☭ ☢ €"
  _post "$message"

  run _search "$message"
  [[ "$status" -eq 0 ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')" == *"$message"* ]]

}