#!/usr/bin/env bats

source tests/helpers.bash

#function setup(){
#  _make deploy_elasticsearch
#}

#function teardown(){
#  _make undeploy_elasticsearch
#}

#@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - all default fields are logged" {
#
#  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
#  message="$((RANDOM)) $BATS_TEST_DESCRIPTION"
#
#  run _dockerRun --rm --name "$name" alpine echo -n "$message"
#  [[ "$status" -eq 0 ]]
#
#  run _get "message:\"$message\""
#  [[ "$status" -eq 0 ]]
#  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '1 p')" == "containerCreated" ]]
#  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '2 p')" == "containerID" ]]
#  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '3 p')" == "containerImageName" ]]
#  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '4 p')" == "containerName" ]]
#  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '5 p')" == "message" ]]
#  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '6 p')" == "partial" ]]
#  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '7 p')" == "source" ]]
#  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | sed -n '8 p')" == "timestamp" ]]
#  [[ "$(echo ${output} | jq '.hits.hits[0]._source' | jq -r 'keys[]' | wc -l)" -eq 8 ]]
#
#}

function test_one(){
    echo "[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - all default fields are filled out"

  basht_run make -f "$MAKEFILE" deploy_elasticsearch
  _getProtocol &>/dev/null
  _elasticsearchHealth &>/dev/null

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  name="test.2"
  message="$((RANDOM)) da"

  basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    alpine echo -n "$message"

sleep 2

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""
  [[ "$status" -eq 0 ]]
#   basht_assert "$(echo ${output} | jq -r '.hits.hits[0]._source.containerCreated')"   =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]+Z$
#   basht_assert "$(echo ${output} | jq -r '.hits.hits[0]._source.containerID')"        =~ ^[a-z0-9]{12}$
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.containerImageName'" == "alpine"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.containerName'"      == "$name"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.message'"            == "$message"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.source'"             == "stdout"
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.partial'"            == "true"
#   basht_assert "$(echo ${output} | jq -r '.hits.hits[0]._source.timestamp')"          =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]+Z$
#   basht_assert "$(echo ${output} | jq -r '.hits.hits[0]._source[]' | wc -l)" -eq 8
  basht_run make -f "$MAKEFILE" undeploy_elasticsearch
}