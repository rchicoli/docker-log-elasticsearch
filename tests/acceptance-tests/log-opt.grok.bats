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
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.auth')"        == "-"         ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.bytes')"       == "207"       ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.clientip')"    == "127.0.0.1" ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.httpversion')" == "1.1"   ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.ident')"       == "-"     ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.rawrequest')"  == ""      ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.response')"    == "404"   ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.timestamp')"   == "23/Apr/2014:22:58:32 +0200" ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.verb')"        == "GET"   ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok[]' | wc -l)" -eq 10 ]]

}
