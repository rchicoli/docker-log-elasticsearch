#!/bin/bash

source tests/helpers.bash

function setUp(){
  _make deploy_elasticsearch 1>/dev/null
  _getProtocol 1>/dev/null
  _elasticsearchHealth 1>/dev/null
}

function tearDown(){
  _debug
  _make undeploy_nginx 1>/dev/null
  _make undeploy_elasticsearch 1>/dev/null
  docker system prune -f 1>/dev/null
}

#@test "[${BATS_TEST_FILENAME##*/}] integration-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - container starts using https protocol" {
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
#  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
#  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"
#
#  run _dockerRun --rm --name $name \
#    --log-opt elasticsearch-sniff='false' \
#    --log-opt elasticsearch-username=${ELASTICSEARCH_USERNAME:-elastic} \
#    --log-opt elasticsearch-password=${ELASTICSEARCH_PASSWORD:-changeme} \
#    --log-opt elasticsearch-insecure='true' \
#    alpine echo -n "$message"
#
#  [[ "$status" -eq 0 ]]
#
#}

function test_container_start_using_https_via_proxy(){

  description="[${BASHT_TEST_FILENAME##*/}] integration-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - container starts using a proxy behind elasticsearch"
  echo "$description"

  [[ ${CLIENT_VERSION} -eq 1 ]] && return 0 # "todo: if required"

  #if [[ ${CLIENT_VERSION} -eq 6 ]]; then
  #  basht_run ${SCRIPTS_DIR}/wait-for-it.sh elasticsearch 9200 echo wait before setting up a password
  #  basht_run export ELASTICSEARCH_PASSWORD="`docker exec -ti elasticsearch bash -c './bin/x-pack/setup-passwords auto --batch' | awk '/PASSWORD elastic/ {print $4}' | tr -d '[:space:]'`"
  #fi
  basht_run _make deploy_nginx

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) $description"

  ${SCRIPTS_DIR}/wait-for-it.sh nginx 443 echo wait until proxy is ready

  export ELASTICSEARCH_IP=172.31.0.4
  export ELASTICSEARCH_PORT=443
  basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt elasticsearch-username=${ELASTICSEARCH_USERNAME:-elastic} \
    --log-opt elasticsearch-password=${ELASTICSEARCH_PASSWORD:-changeme} \
    --log-opt elasticsearch-insecure='true' \
    --log-opt elasticsearch-sniff='false' \
    alpine echo -n "$message"

}