#!/usr/bin/env bats

load ../helpers

function setup(){
  _make deploy_elasticsearch
}

function teardown(){
  _make undeploy_elasticsearch
}

@test "[${BATS_TEST_FILENAME##*/}] acceptance-tests (v${CLIENT_VERSION}): $BATS_TEST_NUMBER - java exception are merged into one line" {

  name="${BATS_TEST_FILENAME##*/}.${BATS_TEST_NUMBER}"
  message="$((RANDOM)) $BATS_TEST_DESCRIPTION 2018-03-15 16:06:00,011 ERROR [task-scheduler-9] [LoggingHandler.java:145] org.springframework.integration.handler.ReplyRequiredException: No reply produced by handler 'prepareJobLaunchRequest', and its 'requiresReply' property is set to true.
        at org.springframework.integration.handler.AbstractReplyProducingMessageHandler.handleMessageInternal(AbstractReplyProducingMessageHandler.java:180)
        at org.springframework.integration.handler.AbstractMessageHandler.handleMessage(AbstractMessageHandler.java:78)
        at org.springframework.integration.dispatcher.AbstractDispatcher.tryOptimizedDispatch(AbstractDispatcher.java:116)
        at java.lang.Thread.run(Thread.java:745)
  notmachted"

  run _dockerRun --rm --name $name \
    --log-opt elasticsearch-bulk-workers=2 \
    --log-opt elasticsearch-bulk-actions=2 \
    --log-opt elasticsearch-bulk-size="-1" \
    --log-opt elasticsearch-bulk-flush-interval=1s \
    --log-opt elasticsearch-bulk-stats=false \
    alpine echo -n "$message"
  [[ "$status" -eq 0 ]] || _debug "$output"

  run _get "message:\"$message\""
  [[ "$status" -eq 0 ]] || _debug "$output"
  [[ "$(echo ${output} | jq -r '.hits.hits[0]._source.message')" == "$message" ]] || _debug "$output"

}
