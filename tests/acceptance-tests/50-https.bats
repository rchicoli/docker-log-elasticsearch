#!/usr/bin/env bats

load ../helpers

function teardown(){
  [[ ${CLIENT_VERSION} -eq 1 ]] && return 0
  _make undeploy_nginx
  _make undeploy_elasticsearch
}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - https protocol is supported" {

  [[ ${CLIENT_VERSION} -eq 1 ]] && skip "todo: if required"

  export TLS="true"
  _make deploy_elasticsearch

  if [[ ${CLIENT_VERSION} -eq 6 ]]; then
    ${SCRIPTS_DIR}/wait-for-it.sh elasticsearch 9200 echo wait before setting up a password
    export ELASTICSEARCH_PASSWORD="`docker exec -ti elasticsearch bash -c './bin/x-pack/setup-passwords auto --batch' | awk '/PASSWORD elastic/ {print $4}' | tr -d '[:space:]'`"
  fi

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"

  run _dockerRun --rm --name $name \
    --log-opt elasticsearch-sniff='false' \
    --log-opt elasticsearch-username=${ELASTICSEARCH_USERNAME:-elastic} \
    --log-opt elasticsearch-password=${ELASTICSEARCH_PASSWORD:-changeme} \
    --log-opt elasticsearch-insecure='true' \
    alpine echo -n "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"

  run _get "message:\"$message\""
  [[ "$status" -eq 0 ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')" == "$message" ]] || _debug "$output"

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - https protocol works through the proxy" {

  _make deploy_elasticsearch
  if [[ ${CLIENT_VERSION} -eq 6 ]]; then
    ${SCRIPTS_DIR}/wait-for-it.sh elasticsearch 9200 echo wait before setting up a password
    export ELASTICSEARCH_PASSWORD="`docker exec -ti elasticsearch bash -c './bin/x-pack/setup-passwords auto --batch' | awk '/PASSWORD elastic/ {print $4}' | tr -d '[:space:]'`"
  fi

  _make deploy_nginx

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"

  export ELASTICSEARCH_URL="https://172.31.0.4:443"
  run _dockerRun --rm --name $name \
    --log-opt elasticsearch-sniff='false' \
    --log-opt elasticsearch-username=${ELASTICSEARCH_USERNAME:-elastic} \
    --log-opt elasticsearch-password=${ELASTICSEARCH_PASSWORD:-changeme} \
    --log-opt elasticsearch-insecure='true' \
    alpine echo -n "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"

  run _get "message:\"$message\""
  [[ "$status" -eq 0 ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')" == "$message" ]] || _debug "$output"

}