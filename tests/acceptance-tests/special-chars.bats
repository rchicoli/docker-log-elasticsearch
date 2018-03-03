#!/usr/bin/env bats

load ../helpers

function setup(){
  _make create_environment
}

function teardown(){
  _make delete_environment
}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - unicode characters are accepted" {

  message="${BATS_TEST_NUMBER}:héllö-yöü ❤ ☀ ☆ ☂ ☻ ♞ ☯ ☭ ☢ €"
  _post "$message"

  # no idea why travis fails time to time, so we sleep
  sleep 1

  run _search "$message"
  [[ "$status" -eq 0 ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')" == *"$message"* ]]

}
