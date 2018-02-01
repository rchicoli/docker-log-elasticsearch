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

MAKEFILE="${BASE_DIR}/Makefile"

POST_MESSAGE="this-is-a-one-logging-line"

function _post() {
  local id="$1"
  curl "http://${WEBAPPER_IP}:${WEBAPPER_PORT}/${1}" &>/dev/null
  sleep 2
}

function _search() {
    local message="$1"
    curl -s ${ELASTICSEARCH_HTTP_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search\?pretty=true\&size=1\&q=message:\"${message}\"
}

# make wrapper
function _make() {
  make -f "$MAKEFILE" "$@"
}

function _expr() {
  expr "$(cat /dev/stdin)" : "$@"
}