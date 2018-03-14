#!/usr/bin/env bats

load ../helpers

function setup(){
  _make deploy_elasticsearch
}

function teardown(){
  _make undeploy_elasticsearch
}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - unicode characters are accepted" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) ${BATS_TEST_DESCRIPTION}: héllö-yöü ❤ ☀ ☆ ☂ ☻ ♞ ☯ ☭ ☢ €"

  _dockerRun --rm --name $name \
    alpine echo -n "$message"

  run _search "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')" == "$message" ]] || _debug "$output"

}