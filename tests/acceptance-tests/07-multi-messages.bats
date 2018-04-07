#!/usr/bin/env bats

load ../helpers

function setup(){
  _make create_environment
}

function teardown(){
  _make delete_environment
}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - multiple containers with different configurations" {

  message="1 - $((RANDOM)) $BATS_TEST_DESCRIPTION"
  run _post "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"

  run _get "message:\"$message\""
  [[ "$status" -eq 0 ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')" == "$message" ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerName')" == "webapper" ]] || _debug "$output"

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}.$((RANDOM))"
  message="2 - $((RANDOM)) $BATS_TEST_DESCRIPTION"
  run _dockerRun --rm --name $name \
    --log-opt elasticsearch-bulk-workers=2 \
    --log-opt elasticsearch-bulk-actions=2 \
    --log-opt elasticsearch-bulk-size="-1" \
    --log-opt elasticsearch-bulk-flush-interval=1s \
    --log-opt elasticsearch-bulk-stats=false \
    --log-opt elasticsearch-fields='config,containerID,containerName,containerArgs,containerImageID,containerImageName,containerCreated,containerEnv,containerLabels,daemonName' \
    alpine echo -n "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"

  run _get "message:\"$message\""
  [[ "$status" -eq 0 ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')" == "$message" ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerName')" == "$name" ]] || _debug "$output"

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}.$((RANDOM))"
  message="$((RANDOM)) $name"
  run _dockerRun --rm --name $name \
    --log-opt elasticsearch-bulk-workers=2 \
    --log-opt elasticsearch-bulk-actions=2 \
    --log-opt elasticsearch-bulk-size="-1" \
    --log-opt elasticsearch-bulk-flush-interval=1s \
    --log-opt elasticsearch-bulk-stats=false \
    --log-opt grok-pattern='MY_NUMBER=(?:[+-]?(?:[0-9]+)) && MY_USER=[a-zA-Z0-9._-]+ && MY_PATTERN=%{MY_NUMBER:random_number} %{MY_USER:user}' \
    --log-opt grok-pattern-splitter=' && ' \
    --log-opt grok-match='%{MY_PATTERN:line}' \
  alpine echo -n "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"

  run _get "grok.line:\"$message\""
  [[ "$status" -eq 0 ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.line')" == "$message" ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerName')" == "$name" ]] || _debug "$output"

  message="4 - $((RANDOM)) $BATS_TEST_DESCRIPTION"
  run _post "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"

  run _get "message:\"$message\""
  [[ "$status" -eq 0 ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')" == "$message" ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerName')" == "webapper" ]] || _debug "$output"


}