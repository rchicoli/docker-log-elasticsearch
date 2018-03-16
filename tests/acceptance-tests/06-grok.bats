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

  run _dockerRun --rm --name "$name" \
    --log-opt grok-match='%{COMMONAPACHELOG}' \
    alpine echo -n "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"

  run _get "message:\"$message\""
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
  message="$((RANDOM)) failed to parse message"

  run _dockerRun --rm --name "$name" \
    --name ${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER} \
    --log-opt grok-match='wrong %{WORD:test1} %{WORD:test2}' alpine echo -n "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"

  run _get "grok.failed:$message"
  [[ "$status" -eq 0 ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.line')" == "$message" ]] || _debug "$output"
  [[ -n "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.err')" ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok[]' | wc -l)" -eq 2 ]] || _debug "$output"

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - custom grok pattern" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) 127.0.0.1 john"

  run _dockerRun --rm --name "$name" \
    --name ${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER} \
    --log-opt grok-pattern='CUSTOM_IP=(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?) and custom_username=%{USERNAME}' \
    --log-opt grok-match='%{NUMBER:random_number} %{CUSTOM_IP:ipv4} %{custom_username:user}' \
     alpine echo -n "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"

  run _get "grok:$message"
  [[ "$status" -eq 0 ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.ipv4')" == "127.0.0.1" ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.user')" == "john" ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.random_number')" =~ [0-9]+ ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok[]' | wc -l)" -eq 3 ]] || _debug "$output"

}


@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - custom grok pattern with different splitter" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) 127.0.0.2 bob"

  run _dockerRun --rm --name "$name" \
    --name ${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER} \
    --log-opt grok-pattern-splitter=" && " \
    --log-opt grok-pattern='CUSTOM_IP=(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?) && custom_username=%{USERNAME}' \
    --log-opt grok-match='%{NUMBER:random_number} %{CUSTOM_IP:ipv4} %{custom_username:user}' \
     alpine echo -n "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"

  run _get "grok:$message"
  [[ "$status" -eq 0 ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.ipv4')" == "127.0.0.2" ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.user')" == "bob" ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.random_number')" =~ [0-9]+ ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok[]' | wc -l)" -eq 3 ]] || _debug "$output"

}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - grok named capture false" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) 127.0.0.3 tester"

  run _dockerRun --rm --name "$name" \
    --name ${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER} \
    --log-opt grok-pattern='CUSTOM_IP=(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?) and custom_username=%{USERNAME}' \
    --log-opt grok-match='%{NUMBER:random_number} %{CUSTOM_IP:ipv4} %{custom_username:user}' \
    --log-opt grok-named-capture=false \
     alpine echo -n "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"

  run _get "grok:$message"
  [[ "$status" -eq 0 ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.ipv4')" == "127.0.0.3" ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.BASE10NUM')" =~ [0-9]+ ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.USERNAME')" == "tester" ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.user')" == "tester" ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok.random_number')" =~ [0-9]+ ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.grok[]' | wc -l)" -eq 5 ]] || _debug "$output"

}
