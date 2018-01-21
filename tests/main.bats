#!/usr/bin/env bats

load helpers

function setup(){
  _make create_environment
}

function teardown(){
  _make delete_environment
}

@test "send log message to elasticsearch" {

  sample_message="this-is-one-logging-line"
  curl "http://${WEBAPPER_IP}:${WEBAPPER_PORT}/$sample_message" &>/dev/null
  sleep 2

  run curl -s http://${ELASTICSEARCH_IP}:${ELASTICSEARCH_PORT}/docker-compose/ci/_search\?pretty=true\&size=1\&q=message:\"$sample_message\"
  [[ "$status" -eq 0 ]]

  [[ "$(echo ${output} | _jq 'containerID'       |  _expr  '[a-z0-9]*')"           -eq 12  ]]
  [[ "$(echo ${output} | _jq 'containerImageID'  |  _expr  'sha256:[a-z0-9]*')"    -ne 72  ]]
  [[ "$(echo ${output} | _jq 'message'           |  _expr  ".*${sample_message}")" -ne 46  ]]

  [[ "$(echo ${output} | _jq 'containerName')"      ==  "webapper"           ]]
  [[ "$(echo ${output} | _jq 'containerImageName')" ==  "rchicoli/webapper"  ]]
  [[ "$(echo ${output} | _jq 'source')"             ==  "stderr"             ]]
  [[ "$(echo ${output} | _jq 'partial')"            ==  "false"              ]]

}
