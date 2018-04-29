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

function test_bulk_commit_after_one_action(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - bulk commit after one action"
  echo "$description"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) $description"

    basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt elasticsearch-bulk-workers=2 \
    --log-opt elasticsearch-bulk-actions=1 \
    --log-opt elasticsearch-bulk-flush-interval=30s \
    alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.message'" == "$message"

}

function test_bulk_disable_actions_and_bulk_size(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - bulk disable actions and bulk size"
  echo "$description"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) $description"

    basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt elasticsearch-bulk-workers=1 \
    --log-opt elasticsearch-bulk-actions="-1" \
    --log-opt elasticsearch-bulk-size="-1" \
    --log-opt elasticsearch-bulk-flush-interval="10s" \
    alpine echo -n "$message"

  # total numbers of hits should be zero, because the flush interval has not been reached
  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""
  basht_assert "echo '${output}' | jq -r '.hits.total'" == 0

  sleep 10s

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.message'" == "$message"

}