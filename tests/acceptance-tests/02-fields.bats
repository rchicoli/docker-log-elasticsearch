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
  [[ "$status" -eq 0 ]]
  [[ "${lines[0]}" == "message" ]]
  [[ "${lines[1]}" == "partial" ]]
  [[ "${lines[2]}" == "source" ]]
  [[ "${lines[3]}" == "timestamp" ]]
  [[ "${#lines[@]}" -eq 4 ]]

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - all fields are logged, except of containerLabels" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"

  _dockerRun --rm --name $name \
    --log-opt elasticsearch-fields='config,containerID,containerName,containerArgs,containerImageID,containerImageName,containerCreated,containerEnv,daemonName' \
    alpine echo -n "$message"

  run _fields "$message"
  [[ "$status" -eq 0 ]]
  [[ "${lines[0]}" == "config" ]]
  [[ "${lines[1]}" == "containerArgs" ]]
  [[ "${lines[2]}" == "containerCreated" ]]
  [[ "${lines[3]}" == "containerEnv" ]]
  [[ "${lines[4]}" == "containerID" ]]
  [[ "${lines[5]}" == "containerImageID" ]]
  [[ "${lines[6]}" == "containerImageName" ]]
  [[ "${lines[7]}" == "containerName" ]]
  [[ "${lines[8]}" == "daemonName" ]]
  [[ "${lines[9]}" == "message" ]]
  [[ "${lines[10]}" == "partial" ]]
  [[ "${lines[11]}" == "source" ]]
  [[ "${lines[12]}" == "timestamp" ]]
  [[ "${#lines[@]}" -eq 13 ]]

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
  [[ "$status" -eq 0 ]]
  [[ "${lines[0]}" == "config" ]]
  [[ "${lines[1]}" == "containerArgs" ]]
  [[ "${lines[2]}" == "containerCreated" ]]
  [[ "${lines[3]}" == "containerEnv" ]]
  [[ "${lines[4]}" == "containerID" ]]
  [[ "${lines[5]}" == "containerImageID" ]]
  [[ "${lines[6]}" == "containerImageName" ]]
  [[ "${lines[7]}" == "containerLabels" ]]
  [[ "${lines[8]}" == "containerName" ]]
  [[ "${lines[9]}" == "daemonName" ]]
  [[ "${lines[10]}" == "message" ]]
  [[ "${lines[11]}" == "partial" ]]
  [[ "${lines[12]}" == "source" ]]
  [[ "${lines[13]}" == "timestamp" ]]
  [[ "${#lines[@]}" -eq 14 ]]

}
