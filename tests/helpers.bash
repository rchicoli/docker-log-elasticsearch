#!/bin/bash

ELASTICSEARCH_IP="172.31.0.2"
ELASTICSEARCH_PORT="9200"

ELASTICSEARCH_INDEX="docker"
ELASTICSEARCH_TYPE="log"

WEBAPPER_IP="172.31.0.3"
WEBAPPER_PORT="8080"

# this is required for the makefile
export BASE_DIR="$BATS_TEST_DIRNAME/../.."
export CLIENT_VERSION="${CLIENT_VERSION:-5}"

DOCKER_COMPOSE_DIR="${BASE_DIR}/docker"
SCRIPTS_DIR="${BASE_DIR}/scripts"

DOCKER_COMPOSE_FILE="${DOCKER_COMPOSE_DIR}/docker-compose.yml"

ELASTICSEARCH_HTTP_URL="http://${ELASTICSEARCH_IP}:${ELASTICSEARCH_PORT}"
ELASTICSEARCH_HTTPS_URL="https://${ELASTICSEARCH_IP}:${ELASTICSEARCH_PORT}"

function _getProtocol(){
  ELASTICSEARCH_URL="$ELASTICSEARCH_HTTP_URL"
  if [[ "$TLS" == "true" ]]; then
    ELASTICSEARCH_URL="$ELASTICSEARCH_HTTPS_URL"
  fi
}

ELASTICSEARCH_USERNAME="${ELASTICSEARCH_USERNAME:-elastic}"
ELASTICSEARCH_PASSWORD="${ELASTICSEARCH_PASSWORD:-changeme}"

MAKEFILE="${BASE_DIR}/Makefile"

function _dockerRun(){
  docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url=http://${ELASTICSEARCH_IP}:${ELASTICSEARCH_PORT} \
    "$@"
}

function _post() {
  local id="$1"
  curl -s -XPOST -H "Content-Type: application/json" --data "{\"message\":\"$1\"}" "http://${WEBAPPER_IP}:${WEBAPPER_PORT}/log" &>/tmp/test.log
  # wait 5 seconds until the message can be processed, just in case if there is a system load
  sleep 5
}

function _search() {
    _getProtocol
    local message="$1"
    curl -G -s -k -u "${ELASTICSEARCH_USERNAME}:${ELASTICSEARCH_PASSWORD}" ${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search\?pretty=true\&size=1 --data-urlencode "q=message:\"${message}\""
}

function _curl() {
    _getProtocol
    local message="$1"
    curl -G -s -k -u "${ELASTICSEARCH_USERNAME}:${ELASTICSEARCH_PASSWORD}" ${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search\?pretty=true\&size=1 --data-urlencode "q=${message}"
}

function _fields() {
    _getProtocol
    local message="$1"
    curl -G -s -k -u "${ELASTICSEARCH_USERNAME}:${ELASTICSEARCH_PASSWORD}" ${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search\?pretty=true\&size=1 --data-urlencode "q=message:\"${message}\"" | jq '.hits.hits[0]._source' | jq -r 'keys[]'
}

# make wrapper
function _make() {
  make -f "$MAKEFILE" "$@"
}
