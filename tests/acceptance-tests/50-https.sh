#!/bin/bash

source tests/helpers.bash

function setUp(){
  _make deploy_elasticsearch 1>/dev/null
  _getProtocol 1>/dev/null
  _elasticsearchHealth 1>/dev/null
}

function tearDown(){
  #[[ ${CLIENT_VERSION} -eq 1 ]] && return 0
  _debug
  _make undeploy_nginx
  _make undeploy_elasticsearch
   docker system prune -f
}

#@test "[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - https protocol is supported" {
#
#  [[ ${CLIENT_VERSION} -eq 1 ]] && skip "todo: if required"
#
#  export TLS="true"
#  _make deploy_elasticsearch
#
#  if [[ ${CLIENT_VERSION} -eq 6 ]]; then
#    ${SCRIPTS_DIR}/wait-for-it.sh elasticsearch 9200 echo wait before setting up a password
#    export ELASTICSEARCH_PASSWORD="`docker exec -ti elasticsearch bash -c './bin/x-pack/setup-passwords auto --batch' | awk '/PASSWORD elastic/ {print $4}' | tr -d '[:space:]'`"
#  fi
#
#  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
#  message="$((RANDOM)) $BASHT_TEST_DESCRIPTION"
#
#  run _dockerRun --rm --name $name \
#    --log-opt elasticsearch-sniff='false' \
#    --log-opt elasticsearch-username=${ELASTICSEARCH_USERNAME:-elastic} \
#    --log-opt elasticsearch-password=${ELASTICSEARCH_PASSWORD:-changeme} \
#    --log-opt elasticsearch-insecure='true' \
#    alpine echo -n "$message"
#  [[ "$status" -eq 0 ]]
#
#  run _get "message:\"$message\""
#  [[ "$status" -eq 0 ]]
#  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')" == "$message" ]]
#
#}

function test_https_protocol_via_proxy(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - https protocol works through the proxy"
  echo "$description"

  #if [[ ${CLIENT_VERSION} -eq 6 ]]; then
  #  basht_run ${SCRIPTS_DIR}/wait-for-it.sh elasticsearch 9200 echo wait before setting up a password
  #  basht_run export ELASTICSEARCH_PASSWORD="`docker exec -ti elasticsearch bash -c './bin/x-pack/setup-passwords auto --batch' | awk '/PASSWORD elastic/ {print $4}' | tr -d '[:space:]'`"
  #  [[ "$status" -eq 0 ]]
  #fi
  basht_run _make deploy_nginx

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) $description"

  basht_run ${SCRIPTS_DIR}/wait-for-it.sh nginx 443 echo wait until proxy is ready

  export ELASTICSEARCH_URL="https://172.31.0.4:443"
  basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt elasticsearch-sniff='false' \
    --log-opt elasticsearch-username=${ELASTICSEARCH_USERNAME:-elastic} \
    --log-opt elasticsearch-password=${ELASTICSEARCH_PASSWORD:-changeme} \
    --log-opt elasticsearch-insecure='true' \
    alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  basht_run curl -k -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""

  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.message'" == "$message"

}