#!/usr/bin/env bats

load ../helpers

function setup(){
  _make deploy_elasticsearch
}

function teardown(){
  _make undeploy_elasticsearch
}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - line can be parsed with grok" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] \"GET /index.php HTTP/1.1\" 404 $((RANDOM))"

  _dockerRun --rm --name $name \
    --log-opt grok-pattern='%{COMMONAPACHELOG}' \
    alpine echo -n "$message"

  run _search "$message"
  [[ "$status" -eq 0 ]] || (echo -n "${output}" && 	docker logs elasticsearch && return 1)
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.auth')"        == "-"         ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.bytes')"       =~ [0-9]+      ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.clientip')"    == "127.0.0.1" ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.httpversion')" == "1.1"   ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.ident')"       == "-"     ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.rawrequest')"  == ""      ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.response')"    == "404"   ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.timestamp')"   == "23/Apr/2014:22:58:32 +0200" ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.verb')"        == "GET"   ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok[]' | wc -l)" -eq 10 ]]

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - failed parsed lines are logged" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"
  message="$((RANDOM)) failed to parse message"

  _dockerRun --rm --name $name \
    --name ${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER} \
    --log-opt grok-pattern='wrong %{WORD:test1} %{WORD:test2}' alpine echo -n "$message"

  run _curl "grok.failed:$message"
  [[ "$status" -eq 0 ]] || (echo -n "${output}" && 	docker logs elasticsearch && return 1)
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.failed')" == "$message" ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok[]' | wc -l)" -eq 1 ]]

}
