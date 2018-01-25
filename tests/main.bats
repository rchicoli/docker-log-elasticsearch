#!/usr/bin/env bats

load helpers


@test "send log message to elasticsearch" {

  run bats "${BATS_TEST_DIRNAME}/acceptance-tests"
  [[ "$status" -eq 0 ]]
  [[ "$output" = "1..0" ]]
}
