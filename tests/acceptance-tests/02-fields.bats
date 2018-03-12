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

  _dockerRun --rm --name $name \
    --log-opt elasticsearch-fields='none' \
    alpine echo -n "$message"

  run _fields "$message"
  [[ "$status" -eq 0 ]]              || _debug "$output"
  [[ "${lines[0]}" == "message" ]]   || _debug "$output"
  [[ "${lines[1]}" == "partial" ]]   || _debug "$output"
  [[ "${lines[2]}" == "source" ]]    || _debug "$output"
  [[ "${lines[3]}" == "timestamp" ]] || _debug "$output"
  [[ "${#lines[@]}" -eq 4 ]]         || _debug "$output"

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - all fields are logged, except of containerLabels" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"

  _dockerRun --rm --name $name \
    --log-opt elasticsearch-fields='config,containerID,containerName,containerArgs,containerImageID,containerImageName,containerCreated,containerEnv,daemonName' \
    alpine echo -n "$message"

  run _fields "$message"
  [[ "$status" -eq 0 ]]                       || _debug "$output"
  [[ "${lines[0]}" == "config" ]]             || _debug "$output"
  [[ "${lines[1]}" == "containerArgs" ]]      || _debug "$output"
  [[ "${lines[2]}" == "containerCreated" ]]   || _debug "$output"
  [[ "${lines[3]}" == "containerEnv" ]]       || _debug "$output"
  [[ "${lines[4]}" == "containerID" ]]        || _debug "$output"
  [[ "${lines[5]}" == "containerImageID" ]]   || _debug "$output"
  [[ "${lines[6]}" == "containerImageName" ]] || _debug "$output"
  [[ "${lines[7]}" == "containerName" ]]      || _debug "$output"
  [[ "${lines[8]}" == "daemonName" ]]         || _debug "$output"
  [[ "${lines[9]}" == "message" ]]            || _debug "$output"
  [[ "${lines[10]}" == "partial" ]]           || _debug "$output"
  [[ "${lines[11]}" == "source" ]]            || _debug "$output"
  [[ "${lines[12]}" == "timestamp" ]]         || _debug "$output"
  [[ "${#lines[@]}" -eq 13 ]]                 || _debug "$output"

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - all available fields are logged" {

  # TODO: label is not support on elasticsearch version 2, because of the dots "com.docker.compose.container-number"
  # [[ ${CLIENT_VERSION} -eq 2 ]] && skip "MapperParsingException[Field name [com.docker.compose.config-hash] cannot contain '.']"

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"

  _dockerRun --rm --name $name \
    --log-opt elasticsearch-fields='config,containerID,containerName,containerArgs,containerImageID,containerImageName,containerCreated,containerEnv,containerLabels,daemonName' \
    --label environment=testing \
    alpine echo -n "$message"

  run _fields "$message"
  [[ "$status" -eq 0 ]]                       || _debug "$output"
  [[ "${lines[0]}" == "config" ]]             || _debug "$output"
  [[ "${lines[1]}" == "containerArgs" ]]      || _debug "$output"
  [[ "${lines[2]}" == "containerCreated" ]]   || _debug "$output"
  [[ "${lines[3]}" == "containerEnv" ]]       || _debug "$output"
  [[ "${lines[4]}" == "containerID" ]]        || _debug "$output"
  [[ "${lines[5]}" == "containerImageID" ]]   || _debug "$output"
  [[ "${lines[6]}" == "containerImageName" ]] || _debug "$output"
  [[ "${lines[7]}" == "containerLabels" ]]    || _debug "$output"
  [[ "${lines[8]}" == "containerName" ]]      || _debug "$output"
  [[ "${lines[9]}" == "daemonName" ]]         || _debug "$output"
  [[ "${lines[10]}" == "message" ]]           || _debug "$output"
  [[ "${lines[11]}" == "partial" ]]           || _debug "$output"
  [[ "${lines[12]}" == "source" ]]            || _debug "$output"
  [[ "${lines[13]}" == "timestamp" ]]         || _debug "$output"
  [[ "${#lines[@]}" -eq 14 ]]                 || _debug "$output"

}
