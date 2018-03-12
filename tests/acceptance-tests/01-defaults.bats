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
  [[ "$status" -eq 0 ]]                       || _debug "$output"
  [[ "${lines[0]}" == "containerCreated" ]]   || _debug "$output"
  [[ "${lines[1]}" == "containerID" ]]        || _debug "$output"
  [[ "${lines[2]}" == "containerImageName" ]] || _debug "$output"
  [[ "${lines[3]}" == "containerName" ]]      || _debug "$output"
  [[ "${lines[4]}" == "message" ]]            || _debug "$output"
  [[ "${lines[5]}" == "partial" ]]            || _debug "$output"
  [[ "${lines[6]}" == "source" ]]             || _debug "$output"
  [[ "${lines[7]}" == "timestamp" ]]          || _debug "$output"
  [[ ${#lines[@]} -eq 8 ]]                    || _debug "$output"

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - all default fields are filled out" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"

  _dockerRun --rm --name $name \
    alpine echo -n "$message"

  run _search "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerCreated')"   =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]+Z$ ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerID')"        =~ ^[a-z0-9]{12}$ ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerImageName')" == "alpine"       ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerName')"      == "$name"        ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')"            == "$message"     ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.source')"             == "stdout"       ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.partial')"            == "true"         ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.timestamp')"          =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]+Z$ ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source[]' | wc -l)" -eq 8 ]] || _debug "$output"

}