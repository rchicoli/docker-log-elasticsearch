#!/bin/bash

source tests/helpers.bash

function setUp(){
  _make create_environment 1>/dev/null
  _getProtocol 1>/dev/null
  _elasticsearchHealth 1>/dev/null
}

function tearDown(){
  _debug
  _make delete_environment 1>/dev/null
}

function test_multiple_containers_with_different_configurations(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - multiple containers with different configurations"
  echo "$description"

  message="1 - $((RANDOM)) $description"
  basht_run curl -XPOST -H "Content-Type: application/json" --data "{\"message\":\"$message\"}" "http://${WEBAPPER_IP}:${WEBAPPER_PORT}/log"

  sleep "${SLEEP_TIME}"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.message'"       == "$message"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.containerName'" == "webapper"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}.$((RANDOM))"
  message="2 - $((RANDOM)) $description"
  basht_run _dockerRun --rm --name $name \
    --log-opt elasticsearch-bulk-workers=2 \
    --log-opt elasticsearch-bulk-actions=2 \
    --log-opt elasticsearch-bulk-size="-1" \
    --log-opt elasticsearch-bulk-flush-interval=1s \
    --log-opt elasticsearch-bulk-stats=false \
    --log-opt elasticsearch-fields='config,containerID,containerName,containerArgs,containerImageID,containerImageName,containerCreated,containerEnv,containerLabels,daemonName' \
    alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.message'"       == "$message"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.containerName'" == "$name"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}.$((RANDOM))"
  message="$((RANDOM)) $name"
  basht_run _dockerRun --rm --name $name \
    --log-opt elasticsearch-bulk-workers=2 \
    --log-opt elasticsearch-bulk-actions=2 \
    --log-opt elasticsearch-bulk-size="-1" \
    --log-opt elasticsearch-bulk-flush-interval=1s \
    --log-opt elasticsearch-bulk-stats=false \
    --log-opt grok-pattern='MY_NUMBER=(?:[+-]?(?:[0-9]+)) && MY_USER=[a-zA-Z0-9._-]+ && MY_PATTERN=%{MY_NUMBER:random_number} %{MY_USER:user}' \
    --log-opt grok-pattern-splitter=' && ' \
    --log-opt grok-match='%{MY_PATTERN:line}' \
  alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=grok.line:\"$message\""

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.line'"     == "$message"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.containerName'" == "$name"

  basht_run docker restart webapper

  message="4 - $((RANDOM)) $description"
  basht_run curl -XPOST -H "Content-Type: application/json" --data "{\"message\":\"$message\"}" "http://${WEBAPPER_IP}:${WEBAPPER_PORT}/log"

  sleep "${SLEEP_TIME}"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.message'" == "$message"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.containerName'" == "webapper"

  basht_run curl -XGET -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=100"

  basht_assert "echo '${output}' | jq -r '.hits.total'" == 4

}