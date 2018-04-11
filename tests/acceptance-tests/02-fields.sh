#!/bin/bash

source tests/helpers.bash

function setUp(){
  _make deploy_elasticsearch 1>/dev/null
  _getProtocol 1>/dev/null
  _elasticsearchHealth 1>/dev/null
}

function tearDown(){
  _debug
  _make undeploy_elasticsearch 1>/dev/null
}

function test_only_static_fields(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - only static fields are logged"
  echo "$description"
  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) $description"

  basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt elasticsearch-fields='none' \
    alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""

  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '1 p'" == "message"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '2 p'" == "partial"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '3 p'" == "source"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '4 p'" == "timestamp"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | wc -l"         == 4

}

function test_all_fields_except_container_labels(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - all fields are logged, except of containerLabels"
  echo "$description"
  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) $description"

  basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt elasticsearch-fields='config,containerID,containerName,containerArgs,containerImageID,containerImageName,containerCreated,containerEnv,daemonName' \
    alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""

  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '1 p'"  == "config"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '2 p'"  == "containerArgs"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '3 p'"  == "containerCreated"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '4 p'"  == "containerEnv"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '5 p'"  == "containerID"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '6 p'"  == "containerImageID"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '7 p'"  == "containerImageName"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '8 p'"  == "containerName"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '9 p'"  == "daemonName"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '10 p'" == "message"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '11 p'" == "partial"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '12 p'" == "source"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '13 p'" == "timestamp"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | wc -l"         == 13

}

function test_all_available_fields(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - all available fields are logged"
  echo "$description"

  # TODO: label is not support on elasticsearch version 2, because of the dots "com.docker.compose.container-number"
  # basht_assert ${CLIENT_VERSION} -eq 2 && skip "MapperParsingException[Field name [com.docker.compose.config-hash] cannot contain '.']"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) $description"

  basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt elasticsearch-fields='config,containerID,containerName,containerArgs,containerImageID,containerImageName,containerCreated,containerEnv,containerLabels,daemonName' \
    --label environment=testing \
    alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""

  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '1 p'"  == "config"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '2 p'"  == "containerArgs"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '3 p'"  == "containerCreated"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '4 p'"  == "containerEnv"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '5 p'"  == "containerID"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '6 p'"  == "containerImageID"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '7 p'"  == "containerImageName"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '8 p'"  == "containerLabels"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '9 p'"  == "containerName"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '10 p'" == "daemonName"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '11 p'" == "message"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '12 p'" == "partial"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '13 p'" == "source"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '14 p'" == "timestamp"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | wc -l"         == 14

}
