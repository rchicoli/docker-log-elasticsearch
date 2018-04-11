#!/bin/bash

source tests/helpers.bash

function setUp(){
  _make create_environment 1>/dev/null
  _getProtocol 1>/dev/null
  _elasticsearchHealth 1>/dev/null
}

function tearDown(){
  _debug
  _make delete_environment 1>/dev/null
}


function test_restarting_container(){

  description="[${BASHT_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BASHT_TEST_NUMBER - restarting a container"
  echo "$description"

  basht_run docker restart webapper

  basht_run docker restart webapper

  basht_run docker restart webapper

}