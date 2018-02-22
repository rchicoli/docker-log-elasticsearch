#!/usr/bin/env bats

load ../helpers

function teardown(){
  _make delete_environment
}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - it is possible to log to a different elasticsearch index and type" {

  export TLS="true"
  export ELASTICSEARCH_URL="$ELASTICSEARCH_HTTPS_URL"
  export DOCKER_LOG_OPTIONS="${DOCKER_COMPOSE_DIR}/log-opt.https.yml"
  _make create_environment

  message="${BATS_TEST_NUMBER} log this and that"
  _post "$message"

  run _search "$message"
  [[ "$status" -eq 0 ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')" == *"$message"* ]]

}