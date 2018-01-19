#!/bin/bash

ELASTICSEARCH_IP="172.31.0.2"
ELASTICSEARCH_PORT="9200"

WEBAPPER_IP="172.31.0.3"
WEBAPPER_PORT="8080"

# this is required for the makefile
export BASE_DIR="$BATS_TEST_DIRNAME/.."

DOCKER_COMPOSE_FILE="${BASE_DIR}/docker/docker-compose.yml"
MAKEFILE="${BASE_DIR}/Makefile"
MAKE="make -f ${MAKEFILE} "

# make wrapper
function _make() {
  run make -f "$MAKEFILE" "$@"
}
