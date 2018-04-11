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

function test_unicode_character(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - unicode characters are accepted"
  echo "$description"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) ${description}: héllö-yöü ❤ ☀ ☆ ☂ ☻ ♞ ☯ ☭ ☢ €"

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

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.message'" == "$message"

}