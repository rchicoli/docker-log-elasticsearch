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
  [[ "$status" -eq 0 ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.auth')"        == "-"         ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.bytes')"       =~ [0-9]+      ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.clientip')"    == "127.0.0.1" ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.httpversion')" == "1.1"   ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.ident')"       == "-"     ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.rawrequest')"  == ""      ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.response')"    == "404"   ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.timestamp')"   == "23/Apr/2014:22:58:32 +0200" ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.verb')"        == "GET"   ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok[]' | wc -l)" -eq 10 ]] || _debug "$output"

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - failed parsed lines are logged" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"
  message="$((RANDOM)) failed to parse message"

  _dockerRun --rm --name $name \
    --name ${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER} \
    --log-opt grok-pattern='wrong %{WORD:test1} %{WORD:test2}' alpine echo -n "$message"

  run _curl "grok.failed:$message"
  [[ "$status" -eq 0 ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.failed')" == "$message" ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok[]' | wc -l)" -eq 1 ]] || _debug "$output"

}
