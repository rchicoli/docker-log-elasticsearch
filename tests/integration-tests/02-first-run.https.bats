#!/usr/bin/env bats

load ../helpers

function teardown(){
  _make undeploy_nginx
  _make undeploy_elasticsearch
  docker system prune -f
}

@test "[${BATS_TEST_FILENAME##*/}] integration-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - container starts using https protocol" {

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

}

@test "[${BATS_TEST_FILENAME##*/}] integration-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - container starts using a proxy behind elasticsearch" {

  [[ ${CLIENT_VERSION} -eq 1 ]] && skip "todo: if required"

  _make deploy_elasticsearch
  if [[ ${CLIENT_VERSION} -eq 6 ]]; then
    ${SCRIPTS_DIR}/wait-for-it.sh elasticsearch 9200 echo wait before setting up a password
    export ELASTICSEARCH_PASSWORD="`docker exec -ti elasticsearch bash -c './bin/x-pack/setup-passwords auto --batch' | awk '/PASSWORD elastic/ {print $4}' | tr -d '[:space:]'`"
  fi
  _make deploy_nginx

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"

  ${SCRIPTS_DIR}/wait-for-it.sh nginx 443 echo wait until proxy is ready

  export ELASTICSEARCH_IP=172.31.0.4
  export ELASTICSEARCH_PORT=443
  run _dockerRun --rm --name $name \
    --log-opt elasticsearch-username=${ELASTICSEARCH_USERNAME:-elastic} \
    --log-opt elasticsearch-password=${ELASTICSEARCH_PASSWORD:-changeme} \
    --log-opt elasticsearch-insecure='true' \
    --log-opt elasticsearch-sniff='false' \
    alpine echo -n "$message"

  [[ "$status" -eq 0 ]] || _debug "$output"

}