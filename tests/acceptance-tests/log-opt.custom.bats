#!/usr/bin/env bats

load ../helpers

function teardown(){
  _make delete_environment
}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - it is possible to log to a different elasticsearch index and type" {

  export DOCKER_LOG_OPTIONS="${DOCKER_COMPOSE_DIR}/log-opt.index-and-type.yml"
  export ELASTICSEARCH_INDEX="docker-compose"
  export ELASTICSEARCH_TYPE="ci"
  _make create_environment

  message="${BATS_TEST_NUMBER} log this and that"
  _post "$message"

  run _search "$message"
  [[ "$status" -eq 0 ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')" == *"$message"* ]]

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - all available fields are logged" {

  # TODO: label is not support on elasticsearch version 2, because of the dots "com.docker.compose.container-number"
  # TODO: create another test scenario without labels
  [[ ${CLIENT_VERSION} -eq 2 ]] && skip "MapperParsingException[Field name [com.docker.compose.config-hash] cannot contain '.']"

  export DOCKER_LOG_OPTIONS="${DOCKER_COMPOSE_DIR}/log-opt.all-fields.yml"
  _make create_environment

  message="$BATS_TEST_DESCRIPTION"
  _post "$message"

  run _fields "$message"
  [[ "$status" -eq 0 ]]
  [[ "${lines[0]}" == "config" ]]
  [[ "${lines[1]}" == "containerCreated" ]]
  [[ "${lines[2]}" == "containerEnv" ]]
  [[ "${lines[3]}" == "containerID" ]]
  [[ "${lines[4]}" == "containerImageID" ]]
  [[ "${lines[5]}" == "containerImageName" ]]
  [[ "${lines[6]}" == "containerLabels" ]]
  [[ "${lines[7]}" == "containerName" ]]
  [[ "${lines[8]}" == "daemonName" ]]
  [[ "${lines[9]}" == "message" ]]
  [[ "${lines[10]}" == "partial" ]]
  [[ "${lines[11]}" == "source" ]]
  [[ "${lines[12]}" == "timestamp" ]]
  [[ "${#lines[@]}" -eq 13 ]]

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - all fields are logged, except of containerLabels" {

  export DOCKER_LOG_OPTIONS="${DOCKER_COMPOSE_DIR}/log-opt.fields-without-labels.yml"
  _make create_environment

  message="$BATS_TEST_DESCRIPTION"
  _post "$message"

  run _fields "$message"
  [[ "$status" -eq 0 ]]
  [[ "${lines[0]}" == "config" ]]
  [[ "${lines[1]}" == "containerCreated" ]]
  [[ "${lines[2]}" == "containerEnv" ]]
  [[ "${lines[3]}" == "containerID" ]]
  [[ "${lines[4]}" == "containerImageID" ]]
  [[ "${lines[5]}" == "containerImageName" ]]
  [[ "${lines[6]}" == "containerName" ]]
  [[ "${lines[7]}" == "daemonName" ]]
  [[ "${lines[8]}" == "message" ]]
  [[ "${lines[9]}" == "partial" ]]
  [[ "${lines[10]}" == "source" ]]
  [[ "${lines[11]}" == "timestamp" ]]
  [[ "${#lines[@]}" -eq 12 ]]

}
