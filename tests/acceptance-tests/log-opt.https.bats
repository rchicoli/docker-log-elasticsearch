#!/usr/bin/env bats

load ../helpers

function teardown(){
  _make delete_environment
}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - https protocol is supported" {

  [[ ${CLIENT_VERSION} -eq 1 ]] && skip "todo: if required"

  export TLS="true"
  export ELASTICSEARCH_URL="$ELASTICSEARCH_HTTPS_URL"
  _make deploy_elasticsearch
  if [[ ${CLIENT_VERSION} -eq 6 ]]; then
    ${SCRIPTS_DIR}/wait-for-it.sh elasticsearch 9200 echo wait before setting up a password
    export ELASTICSEARCH_PASSWORD="`docker exec -ti elasticsearch bash -c './bin/x-pack/setup-passwords auto --batch' | awk '/PASSWORD elastic/ {print $4}' | tr -d '[:space:]'`"
  fi

  export DOCKER_LOG_OPTIONS="${DOCKER_COMPOSE_DIR}/log-opt.https.yml"
  _make deploy_webapper

  message="${BATS_TEST_NUMBER} log this and that"
  _post "$message"

  run _search "$message"
  [[ "$status" -eq 0 ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')" == *"$message"* ]]

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - https protocol works through the proxy" {

  _make deploy_elasticsearch
  if [[ ${CLIENT_VERSION} -eq 6 ]]; then
    ${SCRIPTS_DIR}/wait-for-it.sh elasticsearch 9200 echo wait before setting up a password
    export ELASTICSEARCH_PASSWORD="`docker exec -ti elasticsearch bash -c './bin/x-pack/setup-passwords auto --batch' | awk '/PASSWORD elastic/ {print $4}' | tr -d '[:space:]'`"
  fi

  _make deploy_nginx

  export DOCKER_LOG_OPTIONS="${DOCKER_COMPOSE_DIR}/log-opt.https-proxy.yml"
  _make deploy_webapper

  message="${BATS_TEST_NUMBER} log this and that"
  _post "$message"

  run _search "$message"
  [[ "$status" -eq 0 ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')" == *"$message"* ]]

}