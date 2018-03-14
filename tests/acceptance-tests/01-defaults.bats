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

  run _dockerRun --rm --name "$name" alpine echo -n "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"

  run _get "message:\"$message\""
  [[ "$status" -eq 0 ]] || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '1 p')" == "containerCreated" ]]   || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '2 p')" == "containerID" ]]        || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '3 p')" == "containerImageName" ]] || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '4 p')" == "containerName" ]]      || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '5 p')" == "message" ]]            || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '6 p')" == "partial" ]]            || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '7 p')" == "source" ]]             || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '8 p')" == "timestamp" ]]          || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | wc -l)" -eq 8 ]]                    || _debug "$output"

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - all default fields are filled out" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"

  run _dockerRun --rm --name "$name" alpine echo -n "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"

  run _get "message:\"$message\""
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