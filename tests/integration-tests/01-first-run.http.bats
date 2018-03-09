#!/usr/bin/env bats

load ../helpers

function setup(){
  _make deploy_elasticsearch
}

function teardown(){
  _make undeploy_elasticsearch
  docker rm $(docker ps --filter status=exited -q)
}

@test "[${BATS_TEST_FILENAME##*/}] integration-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - create a container with the default logging options" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"

  _dockerRunDefault --name $name \
    alpine echo -n "$message"

  run docker inspect $name
  [[ "$(echo ${output} | jq -r '.[0].HostConfig.LogConfig.Config' | jq -r 'keys[]')" == "elasticsearch-url" ]]
  [[ "$(echo ${output} | jq -r '.[0].HostConfig.LogConfig.Config[]' | wc -l)" -eq 1 ]]

}

@test "[${BATS_TEST_FILENAME##*/}] integration-tests (v${CLIENT_VERSION}): check the elasticsearch-version option for different elasticsearch versions" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"

  _dockerRun --name $name \
    alpine echo -n "$message"

  run docker inspect $name
  [[ "$status" -eq 0 ]]
  [[ "$(echo ${output} | jq -r '.[0].HostConfig.LogConfig.Config[]' | wc -l)" -eq 2 ]]
  [[ "$(echo ${output} | jq -r '.[0].HostConfig.LogConfig.Config."elasticsearch-version"')" -eq "$CLIENT_VERSION" ]]

}
