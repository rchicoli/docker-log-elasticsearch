#!/usr/bin/env bats

load ../helpers

function setup(){
  _make deploy_elasticsearch
}

function teardown(){
  _make undeploy_elasticsearch
}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - all default fields are logged" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"

  _dockerRun --rm --name $name \
    alpine echo -n "$message"

  run _fields "$message"
  [[ "$status" -eq 0 ]] || (echo -n "${output}" && 	docker logs elasticsearch && return 1)
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

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - all default fields are filled out" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"

  _dockerRun --rm --name $name \
    alpine echo -n "$message"

  run _search "$message"
  [[ "$status" -eq 0 ]] || (echo -n "${output}" && 	docker logs elasticsearch && return 1)
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerCreated')"   =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]+Z$ ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerID')"        =~ ^[a-z0-9]{12}$ ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerImageName')" == "alpine"       ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerName')"      == "$name"        ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')"            == "$message"     ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.source')"             == "stdout"       ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.partial')"            == "true"         ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.timestamp')"          =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]+Z$ ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source[]' | wc -l)" -eq 8 ]]

}