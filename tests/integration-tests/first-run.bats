#!/usr/bin/env bats

load ../helpers

function setup(){
  _make create_environment
}

function teardown(){
  _make delete_environment
}

@test  "[${BATS_TEST_FILENAME##*/}] integration-tests (v${CLIENT_VERSION}): create a container with the default logging options" {

  [[ ${CLIENT_VERSION} -ne 5 ]] && skip "this checks the default options which is version 5"

  run docker inspect webapper
  [[ "$status" -eq 0 ]]
  [[ "$(echo ${output} | docker inspect webapper | jq -r '.[0].HostConfig.LogConfig.Config[]' | wc -l)" -eq 1 ]]
  [[ "$(echo ${output} | docker inspect webapper | jq -r '.[0].HostConfig.LogConfig.Config' | jq -r 'keys[]')" == "elasticsearch-url" ]]

}

@test "[${BATS_TEST_FILENAME##*/}] integration-tests (v${CLIENT_VERSION}): check the elasticsearch-version option for different elasticsearch versions" {

  [[ ${CLIENT_VERSION} -eq 5 ]] && skip "elasticsearch version 5 does contain the elasticsearch-version option, because it is the default version"

  run docker inspect webapper
  [[ "$status" -eq 0 ]]
  [[ "$(echo ${output} | docker inspect webapper | jq -r '.[0].HostConfig.LogConfig.Config[]' | wc -l)" -eq 2 ]]
  [[ "$(echo ${output} | docker inspect webapper | jq -r '.[0].HostConfig.LogConfig.Config."elasticsearch-version"')" -eq "$CLIENT_VERSION" ]]

}


@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - container starts using https protocol" {

  [[ ${CLIENT_VERSION} -eq 1 ]] && skip "todo: if required"

  export TLS="true"
  export ELASTICSEARCH_URL="$ELASTICSEARCH_HTTPS_URL"
  _make deploy_elasticsearch
  if [[ ${CLIENT_VERSION} -eq 6 ]]; then
    ${SCRIPTS_DIR}/wait-for-it.sh elasticsearch 9200 echo wait before setting up a password
    export ELASTICSEARCH_PASSWORD="`docker exec -ti elasticsearch bash -c './bin/x-pack/setup-passwords auto --batch' | awk '/PASSWORD elastic/ {print $4}' | tr -d '[:space:]'`"
  fi

  export DOCKER_LOG_OPTIONS="${DOCKER_COMPOSE_DIR}/log-opt.https.yml"
  run _make deploy_webapper
  [[ "$status" -eq 0 ]]

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - container starts using a proxy behind elasticsearch" {

  _make deploy_elasticsearch
  if [[ ${CLIENT_VERSION} -eq 6 ]]; then
    ${SCRIPTS_DIR}/wait-for-it.sh elasticsearch 9200 echo wait before setting up a password
    export ELASTICSEARCH_PASSWORD="`docker exec -ti elasticsearch bash -c './bin/x-pack/setup-passwords auto --batch' | awk '/PASSWORD elastic/ {print $4}' | tr -d '[:space:]'`"
  fi
  _make deploy_nginx

  export DOCKER_LOG_OPTIONS="${DOCKER_COMPOSE_DIR}/log-opt.https-proxy.yml"
  run _make deploy_webapper
  [[ "$status" -eq 0 ]]

}