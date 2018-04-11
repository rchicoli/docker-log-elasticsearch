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

function test_grok_parser(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - line can be parsed with grok"
  echo "$description"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] \"GET /index.php HTTP/1.1\" 404 $((RANDOM))"

  basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt grok-match='%{COMMONAPACHELOG}' \
    --log-opt elasticsearch-bulk-flush-interval=1s \
    alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=grok:\"$message\""

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.auth'"        == "-"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.bytes'"       =~ [0-9]+
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.clientip'"    == "127.0.0.1"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.httpversion'" == "1.1"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.ident'"       == "-"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.rawrequest'"  == ""
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.response'"    == "404"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.timestamp'"   == "23/Apr/2014:22:58:32 +0200"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.verb'"        == "GET"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok[]' | wc -l" == 10

}

function test_failed_parsed_lines_are_logged(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - failed parsed lines are logged"]
  echo "$description"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) failed to parse message"

  basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt elasticsearch-bulk-flush-interval=1s \
    --log-opt grok-match='wrong %{WORD:test1} %{WORD:test2}' alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=grok:'$message'"

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.line'" == "$message"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.err'" == "grok pattern does not match log line"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok[]' | wc -l" == 2

}

function test_custom_grok_pattern(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - custom grok pattern"
  echo "$description"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) 127.0.0.1 john"

  basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name ${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER} \
    --log-opt grok-pattern='CUSTOM_IP=(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?) and custom_username=%{USERNAME}' \
    --log-opt grok-match='%{NUMBER:random_number} %{CUSTOM_IP:ipv4} %{custom_username:user}' \
    --log-opt elasticsearch-bulk-flush-interval=1s \
    alpine echo -n "$message"

  sleep "${SLEEP_TIME}"
  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=grok:'$message'"

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.ipv4'" == "127.0.0.1"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.user'" == "john"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.random_number'" =~ [0-9]+
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok[]' | wc -l" == 3

}

function test_grok_splitter(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - custom grok pattern with different splitter"
  echo "$description"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) 127.0.0.2 bob"

  basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt grok-pattern-splitter=" && " \
    --log-opt grok-pattern='CUSTOM_IP=(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?) && custom_username=%{USERNAME}' \
    --log-opt grok-match='%{NUMBER:random_number} %{CUSTOM_IP:ipv4} %{custom_username:user}' \
    --log-opt elasticsearch-bulk-flush-interval=1s \
    alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=grok:'$message'"

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.ipv4'" == "127.0.0.2"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.user'" == "bob"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.random_number'" =~ [0-9]+
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok[]' | wc -l" == 3

}

function test_grok_named_capture(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - grok named capture false"
  echo "$description"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) 127.0.0.3 tester"

  basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt grok-pattern='CUSTOM_IP=(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?) and custom_username=%{USERNAME}' \
    --log-opt grok-match='%{NUMBER:random_number} %{CUSTOM_IP:ipv4} %{custom_username:user}' \
    --log-opt grok-named-capture=false \
    --log-opt elasticsearch-bulk-flush-interval=1s \
     alpine echo -n "$message"

  sleep "${SLEEP_TIME}"
  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=grok:'$message'"

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.ipv4'" == "127.0.0.3"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.BASE10NUM'" =~ [0-9]+
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.USERNAME'" == "tester"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.user'" == "tester"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.random_number'" =~ [0-9]+
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok[]' | wc -l" == 5

}

function test_grok_pattern_from_file(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - add grok pattern from file"
  echo "$description"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) theo"

  basht_run ${SCRIPTS_DIR}/docker-plugin-folder.sh docker-log-elasticsearch "${CONFIG_DIR}/grok/patterns.txt"

  basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt grok-pattern-from='/tmp/patterns.txt' \
    --log-opt grok-match='%{MY_PATTERN}' \
    --log-opt elasticsearch-bulk-flush-interval=1s \
     alpine echo -n "$message"

  sleep "${SLEEP_TIME}"
  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=grok:'$message'"

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.user'" == "theo"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.random_number'" =~ [0-9]+
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok[]' | wc -l" == 2

}

function test_grok_pattern_from_directory(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - add grok pattern from directory"
  echo "$description"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) max"

  basht_run ${SCRIPTS_DIR}/docker-plugin-folder.sh docker-log-elasticsearch "${CONFIG_DIR}/grok"

  basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt grok-pattern-from='/tmp/grok' \
    --log-opt grok-match='%{MY_PATTERN}' \
    --log-opt elasticsearch-bulk-flush-interval=1s \
     alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=grok:'$message'"

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.user'" == "max"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok.random_number'" =~ [0-9]+
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.grok[]' | wc -l" == 2

}