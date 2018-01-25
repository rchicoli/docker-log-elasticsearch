#!/bin/bash

ELASTICSEARCH_IP="172.31.0.2"
ELASTICSEARCH_PORT="9200"

ELASTICSEARCH_INDEX="docker"
ELASTICSEARCH_TYPE="log"

WEBAPPER_IP="172.31.0.3"
WEBAPPER_PORT="8080"

# this is required for the makefile
export BASE_DIR="$BATS_TEST_DIRNAME/../.."

DOCKER_COMPOSE_DIR="${BASE_DIR}/docker"
SCRIPTS_DIR="${BASE_DIR}/scripts"

DOCKER_COMPOSE_FILE="${DOCKER_COMPOSE_DIR}/docker-compose.yml"
ELASTICSEARCH_V1="${DOCKER_COMPOSE_DIR}/elastic-v1.yml"
ELASTICSEARCH_V2="${DOCKER_COMPOSE_DIR}/elastic-v2.yml"
ELASTICSEARCH_V5="${DOCKER_COMPOSE_DIR}/elastic-v5.yml"

ELASTICSEARCH_HTTP_URL="http://${ELASTICSEARCH_IP}:${ELASTICSEARCH_PORT}"
ELASTICSEARCH_HTTPS_URL="https://${ELASTICSEARCH_IP}:${ELASTICSEARCH_PORT}"

MAKEFILE="${BASE_DIR}/Makefile"
MAKE="make -f ${MAKEFILE} "

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

function _parse_version() {
  version="$1"

  case "$version" in
    "1") echo "${ELASTICSEARCH_V1}"
    ;;
    "2") echo "${ELASTICSEARCH_V2}"
    ;;
    "5") echo "${ELASTICSEARCH_V5}"
    ;;
    *) echo "${ELASTICSEARCH_V5}"
    ;;
  esac

}


function _docker_compose() {
  local app="$1"
  local version="$2"
  version=`_parse_version $version`

  docker-compose -f "$DOCKER_COMPOSE_FILE" -f "$version" up -d "$app"
}

function _app() {
  local app="$1"
  local version="$2"
  version=`_parse_version $version`

 	# "${SCRIPTS_DIR}/wait-for-it.sh" elasticsearch "$ELASTICSEARCH_PORT" "`_docker_compose $app $version`"
 	"${SCRIPTS_DIR}/wait-for-it.sh" elasticsearch  "$ELASTICSEARCH_PORT" docker-compose -f "$DOCKER_COMPOSE_FILE" -f "$version" up -d "$app"
}

# make wrapper
function _make() {
  run make -f "$MAKEFILE" "$@"
}


function _jq() {
  jq -r ".hits.hits[0]._source.${1}"
}

function _expr() {
  expr "$(cat /dev/stdin)" : "$@"
}

# [[ "$(echo ${output} | _jq 'containerImageID'  |  _expr  'sha256:[a-z0-9]*')"    -eq 71  ]]
