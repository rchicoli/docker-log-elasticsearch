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
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.source')" == "stderr" ]]
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.partial')" == "false" ]]
  [[ $(expr "`echo ${output} | jq -r '.hits.hits[0]._source.message'`" : ".*this-is-one-logging-line") -ne 0 ]]

}
