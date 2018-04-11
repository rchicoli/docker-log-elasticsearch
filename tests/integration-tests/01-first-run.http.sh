#!/bin/bash

source tests/helpers.bash

function setUp(){
  _make deploy_elasticsearch 1>/dev/null
  _getProtocol 1>/dev/null
  _elasticsearchHealth 1>/dev/null
}

function tearDown(){
  _debug
  _make undeploy_elasticsearch 1>/dev/null
}

function test_container_with_default_logging(){

  description="[${BASHT_TEST_FILENAME##*/}] integration-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - create a container with the default logging options"
  echo "$description"

  [[ ${CLIENT_VERSION} -ne 5 ]] && echo "this checks the default options which is version 5" && return 0

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) $description"

  basht_run docker run -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --name "$name" alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  basht_run docker inspect "$name"
  basht_assert "echo '${output}' | jq -r '.[0].HostConfig.LogConfig.Config' | jq -r 'keys[]'" == "elasticsearch-url"
  basht_assert "echo '${output}' | jq -r '.[0].HostConfig.LogConfig.Config[]' | wc -l"        == 1

  docker rm "$name"

}

function test_elasticsearch_version_options(){

  description="[${BASHT_TEST_FILENAME##*/}] integration-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - check the elasticsearch-version option for different elasticsearch versions"
  echo "$description"

  name="${BASHT_TEST_FILENAME##*/}.${BASHT_TEST_NUMBER}"
  message="$((RANDOM)) $description"

  basht_run docker run -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    --name "$name" \
    alpine echo -n "$message"

  sleep "${SLEEP_TIME}"

  basht_run docker inspect "$name"
  basht_assert "echo '${output}' | jq -r '.[0].HostConfig.LogConfig.Config[]' | wc -l" == 2
  basht_assert "echo '${output}' | jq -r '.[0].HostConfig.LogConfig.Config.\"elasticsearch-version\"'" == "$CLIENT_VERSION"

  docker rm "$name"

}
