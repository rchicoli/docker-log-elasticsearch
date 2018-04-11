#!/bin/bash

ELASTICSEARCH_IP="172.31.0.2"
ELASTICSEARCH_PORT="9200"

ELASTICSEARCH_INDEX="docker"
ELASTICSEARCH_TYPE="log"

WEBAPPER_IP="172.31.0.3"
WEBAPPER_PORT="8080"

# this is required for the makefile
export BASE_DIR="$BASHT_TEST_DIRNAME/.."
export CLIENT_VERSION="${CLIENT_VERSION:-5}"

DOCKER_COMPOSE_DIR="${BASE_DIR}/docker"
SCRIPTS_DIR="${BASE_DIR}/scripts"
CONFIG_DIR="${BASE_DIR}/config"
TESTS_DIR="${BASE_DIR}/tests"

DOCKER_COMPOSE_FILE="${DOCKER_COMPOSE_DIR}/docker-compose.yml"

ELASTICSEARCH_HTTP_URL="http://${ELASTICSEARCH_IP}:${ELASTICSEARCH_PORT}"
ELASTICSEARCH_HTTPS_URL="https://${ELASTICSEARCH_IP}:${ELASTICSEARCH_PORT}"

function _getProtocol(){
  if [[ "$TLS" == "true" ]] && [[ -z ${ELASTICSEARCH_URL:+x} ]]; then
    ELASTICSEARCH_URL="$ELASTICSEARCH_HTTPS_URL"
  elif [[ -z ${ELASTICSEARCH_URL:+x} ]]; then
    ELASTICSEARCH_URL="$ELASTICSEARCH_HTTP_URL"
  fi
}

ELASTICSEARCH_USERNAME="${ELASTICSEARCH_USERNAME:-elastic}"
ELASTICSEARCH_PASSWORD="${ELASTICSEARCH_PASSWORD:-changeme}"

MAKEFILE="${BASE_DIR}/Makefile"

SLEEP_TIME=${SLEEP_TIME:-1}

function _elasticsearchHealth() {
  color="$(
    wget --no-check-certificate -q --tries 20 --waitretry=1 --retry-connrefused --timeout 5 \
      --user "${ELASTICSEARCH_USERNAME}" --password "${ELASTICSEARCH_PASSWORD}" \
      -O - ${ELASTICSEARCH_URL}/_cluster/health \
      | jq -r '.status'
  )"
  if [[ "$color" =~ (green|yellow) ]]; then
    echo "$(date) elasticsearch cluster is up"
  else
    echo "$(date) timeout: elasticsearch service is not healthy"
    # continue to see what happen
    # exit 2
  fi
}

function _dockerRun(){
  _getProtocol
  _elasticsearchHealth
  docker run -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    "$@"
}

function _post() {
  local message="$1"
  curl -XPOST -H "Content-Type: application/json" --data "{\"message\":\"$message\"}" "http://${WEBAPPER_IP}:${WEBAPPER_PORT}/log"
}

function _get() {
  _getProtocol
  local message="$1"
  # sleep for the flush interval + 5s
  sleep 1
  curl -G -s -k --connect-timeout 5 -u "${ELASTICSEARCH_USERNAME}:${ELASTICSEARCH_PASSWORD}" \
    ${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search\?pretty=true\&size=1 \
    --data-urlencode "q=${message}"
}

function _search() {
  _getProtocol
  # sleep for the flush interval + 5s
  sleep 1
  curl -G -s -k --connect-timeout 5 -u "${ELASTICSEARCH_USERNAME}:${ELASTICSEARCH_PASSWORD}" \
    ${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search\?pretty=true\&size=100
}

# make wrapper
function _make() {
  make -f "$MAKEFILE" "$@"
}

function _debug() {

  echo "$(date) OUTPUT:"

  uptime
  free -m --human
  iostat -x
  docker ps -a
  docker logs elasticsearch

  tail -n50 /var/log/upstart/docker.log || echo "log does not exist"

  echo "searching for all documents"
  curl -k --connect-timeout 5 -u "${ELASTICSEARCH_USERNAME}:${ELASTICSEARCH_PASSWORD}" \
    ${ELASTICSEARCH_URL}/_search\?pretty=true\&size=100

}