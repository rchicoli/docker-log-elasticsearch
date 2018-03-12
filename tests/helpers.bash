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
  if [[ "$TLS" == "true" ]] && [[ -z ${ELASTICSEARCH_URL:+x} ]]; then
    ELASTICSEARCH_URL="$ELASTICSEARCH_HTTPS_URL"
  elif [[ -z ${ELASTICSEARCH_URL:+x} ]]; then
    ELASTICSEARCH_URL="$ELASTICSEARCH_HTTP_URL"
  fi
}

ELASTICSEARCH_USERNAME="${ELASTICSEARCH_USERNAME:-elastic}"
ELASTICSEARCH_PASSWORD="${ELASTICSEARCH_PASSWORD:-changeme}"

MAKEFILE="${BASE_DIR}/Makefile"

function _elasticsearchHealth() {
  color="$(wget --no-check-certificate -q --tries 20 --waitretry=1 --retry-connrefused --timeout 5 --user "${ELASTICSEARCH_USERNAME}" --password "${ELASTICSEARCH_PASSWORD}" -O - ${ELASTICSEARCH_URL}/_cluster/health | jq -r '.status')"
  if [[ "$color" =~ (green|yellow) ]]; then
    echo "$(date) elasticsearch cluster is up"
  else
    echo "$(date) timeout: elasticsearch cluster is not up"
    exit 2
  fi
}

function _retry() {
  local timeout="$1"; shift
  local count=0
  until [[ $("$@" | jq -r '.hits.total' 2>/dev/null) -ne 0 ]]; do
     if [ $count -lt $timeout ]; then
          count=$(($count+1));
      else
          echo "timing out: document not found"
          exit 1
      fi
      sleep 1
  done
  "$@"
}

function _dockerRunDefault(){
  _getProtocol
  _elasticsearchHealth
  sleep 30
  docker run -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    "$@"
}

function _dockerRun(){
  _getProtocol
  _elasticsearchHealth
  sleep 10
  docker run -ti \
    --log-driver rchicoli/docker-log-elasticsearch:development \
    --log-opt elasticsearch-url="${ELASTICSEARCH_URL}" \
    --log-opt elasticsearch-version="${CLIENT_VERSION}" \
    "$@"
}

function _post() {
  local id="$1"
  curl -s -XPOST -H "Content-Type: application/json" --data "{\"message\":\"$id\"}" "http://${WEBAPPER_IP}:${WEBAPPER_PORT}/log"
}

function _search() {
  _getProtocol
  local message="$1"
  sleep 10
  curl -G -s -k --connect-timeout 5 -u "${ELASTICSEARCH_USERNAME}:${ELASTICSEARCH_PASSWORD}" \
    ${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search\?pretty=true\&size=1 \
    --data-urlencode "q=message:\"${message}\""
}


function _curl() {
  _getProtocol
  local message="$1"
  sleep 10
  curl -G -s -k --connect-timeout 5 -u "${ELASTICSEARCH_USERNAME}:${ELASTICSEARCH_PASSWORD}" \
    ${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search\?pretty=true\&size=1 \
    --data-urlencode "q=${message}"

}

function _fields() {
  _getProtocol
  local message="$1"
  sleep 10
  curl -G -s -k --connect-timeout 5 -u "${ELASTICSEARCH_USERNAME}:${ELASTICSEARCH_PASSWORD}" \
    ${ELASTICSEARCH_URL}/${ELASTICSEARCH_INDEX}/${ELASTICSEARCH_TYPE}/_search\?pretty=true\&size=1 \
    --data-urlencode "q=message:\"${message}\"" | jq '.hits.hits[0]._source' | jq -r 'keys[]'

}

# make wrapper
function _make() {
  make -f "$MAKEFILE" "$@"
}

function _getIP() {
  docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$1" 2>/dev/null
}