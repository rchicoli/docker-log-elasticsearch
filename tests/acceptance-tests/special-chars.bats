#!/usr/bin/env bats

load ../helpers

function teardown(){
  _make delete_environment
}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - unicode characters are accepted" {

  _make deploy_elasticsearch
  # no idea why travis fails time to time, so we sleep
  sleep 5
  _make deploy_webapper
  sleep 5

  message="${BATS_TEST_NUMBER}:héllö-yöü ❤ ☀ ☆ ☂ ☻ ♞ ☯ ☭ ☢ €"
  _post "$message"

  # no idea why travis fails time to time, so we sleep
  sleep 5

  run _search "$message"
  [[ "$status" -eq 0 ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')" == *"$message"* ]]

}
