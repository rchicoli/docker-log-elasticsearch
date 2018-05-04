#!/bin/bash

source tests/helpers.bash

function setUp(){
  _make deploy_elasticsearch 1>/dev/null
  _getProtocol 1>/dev/null
  _elasticsearchHealth 1>/dev/null
}

function tearDown(){
  _debug
  basht_run make -f "$MAKEFILE" undeploy_elasticsearch
}

function test_different_index_and_type(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - it is possible to log to a different elasticsearch index and type"
  echo "$description"

  basht_run make -f "$MAKEFILE" deploy_elasticsearch

  export ELASTICSEARCH_INDEX="docker-compose"
  export ELASTICSEARCH_TYPE="ci"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) $description"

  basht_run docker run --rm -ti \
    --log-driver "rchicoli/docker-log-elasticsearch:${PLUGIN_TAG}" \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt elasticsearch-index='docker-compose' \
    --log-opt elasticsearch-type='ci' \
    alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.message'" equals "$message"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.partial'" equals "true"

}

function test_index_regex_year_month_day(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - index regex %F are converted to the current date"
  echo "$description"

  basht_run make -f "$MAKEFILE" deploy_elasticsearch

  export ELASTICSEARCH_INDEX="docker-%F"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) $description"

  basht_run docker run --rm -ti \
    --log-driver "rchicoli/docker-log-elasticsearch:${PLUGIN_TAG}" \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt elasticsearch-index="${ELASTICSEARCH_INDEX}" \
    alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  index_name="$(echo "docker-$(date +%Y.%m.%d)")"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${index_name}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.message'" equals "$message"

}

function test_index_regex_year_month_day(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - all index regex together are converted to the current date"
  echo "$description"

  basht_run make -f "$MAKEFILE" deploy_elasticsearch

  export ELASTICSEARCH_INDEX="test-%b-%B-%d-%j-%m-%y-%y"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) $description"

  basht_run docker run --rm -ti \
    --log-driver "rchicoli/docker-log-elasticsearch:${PLUGIN_TAG}" \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt elasticsearch-index="${ELASTICSEARCH_INDEX}" \
    alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  index_name="$(echo "test-$(date +%b-%B-%d-%j-%m-%y-%y)")"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${index_name,,}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.message'" equals "$message"

}