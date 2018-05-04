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
    --log-driver "rchicoli/docker-log-elasticsearch:${PLUGIN_TAG}" \
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

  docker rm -f "$name"

}

function test_bulk_disable_actions_and_bulk_size(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - bulk disable actions and bulk size"
  echo "$description"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) $description"

    basht_run docker run --rm -ti \
    --log-driver "rchicoli/docker-log-elasticsearch:${PLUGIN_TAG}" \
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

  # wait for the flush interval time
  sleep 10s

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.message'" == "$message"

  docker rm -f "$name"

}

function test_bulk_multiple_messages(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - bulk multiple messages"
  echo "$description"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="bulk-multi-message"

  basht_run docker run -d \
    --log-driver "rchicoli/docker-log-elasticsearch:${PLUGIN_TAG}" \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
     --name "$name" --ip="${WEBAPPER_IP}" --network="docker_development" \
    --log-opt elasticsearch-bulk-actions=100 \
    --log-opt elasticsearch-bulk-flush-interval='2s' \
    --log-opt elasticsearch-bulk-workers=2 \
    rchicoli/webapper

  bulk_size=99

  for i in $(seq 1 "$bulk_size"); do
    basht_run curl -s -XPOST -H "Content-Type: application/json" --data "{\"message\":\"$message-$i\"}" "http://${WEBAPPER_IP}:${WEBAPPER_PORT}/log" >/dev/null
  done

  sleep "${SLEEP_TIME}"

  basht_run docker stop "$name"
  basht_run docker rm "$name"

  sleep "${SLEEP_TIME}"

  basht_run curl -s --connect-timeout 5 "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1"
  basht_assert "echo '${output}' | jq -r '.hits.total'" == "$bulk_size"

  for i in $(seq 1 "$bulk_size"); do
    basht_run curl -G -s --connect-timeout 5 "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
      --data-urlencode "q=message:\"${message}-$i\"" >/dev/null
    basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.message'" equals "$message-$i"
  done

}

# May 03 18:59:34 sunlight dockerd[7729]: time="2018-05-03T18:59:34+02:00" level=error
# msg="level=info msg=\"response error message and status code\"
# containerID=55eeb1ed63dbb828a7bb0ad2a371e1f1f6781b854e8811bd45a4a14ed92f762e
# reason=\"rejected execution of org.elasticsearch.transport.TransportService$7@22386859
# on EsThreadPoolExecutor[bulk, queue capacity = 200,
# org.elasticsearch.common.util.concurrent.EsThreadPoolExecutor@7a781745[Running,
# pool size = 8, active threads = 7, queued tasks = 200, completed tasks = 9740]]\"
# status=429 workerID=77 "
# plugin=5fbed5a261e3721a8260e2aa648ebdc95d2579510faa968c5d4135b8682f7beb
function test_bulk_rejections(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - bulk rejection"
  echo "$description"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="bulk-rejection"

  basht_run docker run -d \
    --log-driver "rchicoli/docker-log-elasticsearch:${PLUGIN_TAG}" \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
     --name "$name" --ip="${WEBAPPER_IP}" --network="docker_development" \
    --log-opt elasticsearch-bulk-actions=500 \
    --log-opt elasticsearch-bulk-flush-interval='5s' \
    --log-opt elasticsearch-bulk-workers=100 \
    rchicoli/webapper

  bulk_size=20000

    seq 1 "$bulk_size" | \
      xargs -n 1 -P 4 \
      curl -s -XPOST -H "Content-Type: application/json" --data "{\"message\":\"$message-$i\"}" "http://${WEBAPPER_IP}:${WEBAPPER_PORT}/log" >/dev/null

  sleep "${SLEEP_TIME}"

  basht_run docker stop "$name"
  basht_run docker rm "$name"

  sleep "${SLEEP_TIME}"

  basht_run curl -s --connect-timeout 5 "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1"
  basht_assert "echo '${output}' | jq -r '.hits.total'" == "$bulk_size"

}
