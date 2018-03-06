#!/usr/bin/env bats

load ../helpers

function teardown(){
  _make delete_environment
}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - line can be parsed with grok" {

  export DOCKER_LOG_OPTIONS="${DOCKER_COMPOSE_DIR}/log-opt.grok.yml"
  _make create_environment

  message='127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] \"GET /index.php HTTP/1.1\" 404 207'
  _post "$message"

  message="127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] \"GET /index.php HTTP/1.1\" 404 207"
  run _search "$message"
  [[ "$status" -eq 0 ]]

  #[[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerCreated' | egrep '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]+Z$')" ]]
  #[[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerID'      | egrep '^[a-z0-9]{12}$')" ]]
  #[[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerImageName')" == "rchicoli/webapper" ]]
  #[[ "$(echo ${output} | jq -r '.hits.hits[0]._source.containerName')"      == "webapper"          ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.COMMONAPACHELOG')"            == *"$message"*        ]]
  #[[ "$(echo ${output} | jq -r '.hits.hits[0]._source.source')"             == "stdout"            ]]
  #[[ "$(echo ${output} | jq -r '.hits.hits[0]._source.partial')"            == "false"             ]]
  #[[ "$(echo ${output} | jq -r '.hits.hits[0]._source.timestamp' | egrep '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]+Z$')" ]]
  #[[ "$(echo ${output} | jq -r '.hits.hits[0]._source[]' | wc -l)" -eq 8 ]]

}
