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

function test_reconnect_after_elasticsearch_restart(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - reconnect after elasticsearch restart"
  echo "$description"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) $description"

  basht_run docker run -d \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
     --name "$name" --ip="${WEBAPPER_IP}" --network="docker_development" \
    --log-opt elasticsearch-bulk-actions=1 \
    --log-opt elasticsearch-bulk-flush-interval='1s' \
    --log-opt elasticsearch-timeout='5s' \
    rchicoli/webapper

  # if elasticsearch is dead, no retry is made

  basht_run docker stop elasticsearch
  basht_run curl -XPOST -H "Content-Type: application/json" --data "{\"message\":\"${message}-1\"}" "http://${WEBAPPER_IP}:${WEBAPPER_PORT}/log"
  basht_run curl -XPOST -H "Content-Type: application/json" --data "{\"message\":\"${message}-2\"}" "http://${WEBAPPER_IP}:${WEBAPPER_PORT}/log"
  basht_run curl -XPOST -H "Content-Type: application/json" --data "{\"message\":\"${message}-3\"}" "http://${WEBAPPER_IP}:${WEBAPPER_PORT}/log"
  basht_run curl -XPOST -H "Content-Type: application/json" --data "{\"message\":\"${message}-4\"}" "http://${WEBAPPER_IP}:${WEBAPPER_PORT}/log"
  basht_run curl -XPOST -H "Content-Type: application/json" --data "{\"message\":\"${message}-5\"}" "http://${WEBAPPER_IP}:${WEBAPPER_PORT}/log"

  # sleep for awhile when elasticsearch is down to test a reconnection
  sleep 10

  basht_run docker start elasticsearch
  basht_run _elasticsearchHealth

  basht_run docker rm -f "$name"
  sleep "${SLEEP_TIME}"

  basht_run curl -s -k -u "${ELASTICSEARCH_USERNAME}:${ELASTICSEARCH_PASSWORD}" --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/_search?pretty=true&size=10"
  basht_assert "echo '${output}' | jq -r '.hits.total'" == 5

}
