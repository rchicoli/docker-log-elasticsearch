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
  [[ "$status" -eq 0 ]]

  run _get "message:\"$message\""
  [[ "$status" -eq 0 ]]
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '1 p')" == "containerCreated" ]]
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '2 p')" == "containerID" ]]
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '3 p')" == "containerImageName" ]]
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '4 p')" == "containerName" ]]
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '5 p')" == "message" ]]
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '6 p')" == "partial" ]]
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '7 p')" == "source" ]]
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '8 p')" == "timestamp" ]]
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | wc -l)" -eq 8 ]]

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - all default fields are filled out" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"

  run _dockerRun --rm --name "$name" alpine echo -n "$message"
  [[ "$status" -eq 0 ]]

  run _get "message:\"$message\""
  [[ "$status" -eq 0 ]]
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