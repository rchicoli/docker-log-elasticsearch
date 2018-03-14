#!/usr/bin/env bats

load ../helpers

function setup(){
  _make deploy_elasticsearch
}

function teardown(){
  _make undeploy_elasticsearch
}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - only static fields are logged" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"

  run _dockerRun --rm --name "$name" \
    --log-opt elasticsearch-fields='none' \
    alpine echo -n "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"

  run _get "message:\"$message\""
  [[ "$status" -eq 0 ]]              || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '1 p')" == "message" ]]   || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '2 p')" == "partial" ]]   || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '3 p')" == "source" ]]    || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '4 p')" == "timestamp" ]] || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | wc -l)" -eq 4 ]]          || _debug "$output"

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - all fields are logged, except of containerLabels" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"

  _dockerRun --rm --name $name \
    --log-opt elasticsearch-fields='config,containerID,containerName,containerArgs,containerImageID,containerImageName,containerCreated,containerEnv,daemonName' \
    alpine echo -n "$message"

  run _get "message:\"$message\""
  [[ "$status" -eq 0 ]]                       || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '1 p')" == "config" ]]             || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '2 p')" == "containerArgs" ]]      || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '3 p')" == "containerCreated" ]]   || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '4 p')" == "containerEnv" ]]       || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '5 p')" == "containerID" ]]        || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '6 p')" == "containerImageID" ]]   || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '7 p')" == "containerImageName" ]] || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '8 p')" == "containerName" ]]      || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '9 p')" == "daemonName" ]]         || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '10 p')" == "message" ]]            || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '11 p')" == "partial" ]]           || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '12 p')" == "source" ]]            || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '13 p')" == "timestamp" ]]         || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | wc -l)" -eq 13 ]]                 || _debug "$output"

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - all available fields are logged" {

  # TODO: label is not support on elasticsearch version 2, because of the dots "com.docker.compose.container-number"
  # [[ ${CLIENT_VERSION} -eq 2 ]] && skip "MapperParsingException[Field name [com.docker.compose.config-hash] cannot contain '.']"

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"

  run _dockerRun --rm --name "$name" \
    --log-opt elasticsearch-fields='config,containerID,containerName,containerArgs,containerImageID,containerImageName,containerCreated,containerEnv,containerLabels,daemonName' \
    --label environment=testing \
    alpine echo -n "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"

  run _get "message:\"$message\""
  [[ "$status" -eq 0 ]]                       || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '1 p')" == "config" ]]             || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '2 p')" == "containerArgs" ]]      || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '3 p')" == "containerCreated" ]]   || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '4 p')" == "containerEnv" ]]       || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '5 p')" == "containerID" ]]        || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '6 p')" == "containerImageID" ]]   || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '7 p')" == "containerImageName" ]] || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '8 p')" == "containerLabels" ]]    || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '9 p')" == "containerName" ]]      || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '10 p')" == "daemonName" ]]         || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '11 p')" == "message" ]]           || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '12 p')" == "partial" ]]           || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '13 p')" == "source" ]]            || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '14 p')" == "timestamp" ]]         || _debug "$output"
  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | wc -l)" -eq 14 ]]                 || _debug "$output"

}
