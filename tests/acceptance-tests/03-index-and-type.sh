#!/usr/bin/env bats

# load ../helpers

source tests/helpers.bash


# function setUp(){
#   _make deploy_elasticsearch
# }

# function tearDown(){
#   _make undeploy_elasticsearch
# }

function test_index_type(){

  echo "[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - it is possible to log to a different elasticsearch index and type"

  basht_run make -f "$MAKEFILE" deploy_elasticsearch
  _getProtocol &>/dev/null
  _elasticsearchHealth &>/dev/null

  export ELASTICSEARCH_INDEX="docker-compose"
  export ELASTICSEARCH_TYPE="ci"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  name="test.1"
  message="a $((RANDOM)) a"

  basht_run docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    --log-opt elasticsearch-index='docker-compose' \
    --log-opt elasticsearch-type='ci' \
    alpine echo -n "a$message"
  # [[ "$status" -eq 1 ]]
#   check_status 0

  sleep 1

  basht_run curl -s -G --connect-timeout 5 \
    "${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search?pretty=true&size=1" \
    --data-urlencode "q=message:\"$message\""
  # [[ "$status" -eq 0 ]]
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.message'" equals "$message"
  # [[ "$status" -eq 0 ]]
  basht_assert "echo '${output}' | jq -r '.hits.hits[0]._source.partial'" equals "true"
  # [[ "$status" -eq 0 ]]

  make -f "$MAKEFILE" undeploy_elasticsearch

}