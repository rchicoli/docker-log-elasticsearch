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

function test_default_fields(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - all default fields are logged"
  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) $description"

  basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""

  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '1 p'" == "containerCreated"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '2 p'" == "containerID"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '3 p'" == "containerImageName"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '4 p'" == "containerName"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '5 p'" == "message"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '6 p'" == "partial"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '7 p'" == "source"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '8 p'" == "timestamp"
  basht_assert "echo '${output}' | jq '.hits.hits[0]._source' | jq -r 'keys[]' | wc -l"        == 8

}

function test_default_fields_are_filled_out(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - all default fields are filled out"
  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) $description"

  basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.containerCreated'"   regexp ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]+Z$
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.containerID'"        regexp ^[a-z0-9]{12}$
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.containerImageName'" == "alpine"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.containerName'"      == "$name"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.message'"            == "$message"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.source'"             == "stdout"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.partial'"            == "true"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.timestamp'"          =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]+Z$
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source[]' | wc -l"          == 8

}