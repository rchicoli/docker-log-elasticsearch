# Docker Log Elasticsearch

`docker-log-elasticsearch` forwards container logs to Elasticsearch service.

This application is under active development and will continue to be modified and improved over time. The current release is an *alpha*." (see [Roadmap](ROADMAP.md)).

## Releases

| Branch Name | Docker Tag | Elasticsearch Version | Remark |
| ----------- | ---------- | --------------------- | ------ |
| master      | 1.0.x      | 1.x, 2.x, 5.x, 6.x    | Future stable release. |
| development | 0.0.1, 0.2.1   | 1.x, 2.x, 5.x, 6.x   | Actively alpha release. |

## Getting Started

You need to install Docker Engine >= 1.12 and Elasticsearch application. Additional information about Docker plugins [can be found here](https://docs.docker.com/engine/extend/plugins_logging/).

### How to install

The following command will download and enable the plugin.

```bash
docker plugin install rchicoli/docker-log-elasticsearch:latest --alias elasticsearch
```

### How to use

#### Prerequisites

Before creating a docker container, a healthy instance of Elasticsearch service must be running.

##### Options #####

| Key | Default Value | Required |
| --- | ------------- | -------- |
| elasticsearch-fields | containerID,containerName,containerImageName,containerCreated | no |
| elasticsearch-index | docker | no  |
| elasticsearch-insecure | false | no |
| elasticsearch-password | no | no |  |
| elasticsearch-sniff | yes | no | |
| elasticsearch-timeout | 1    | no  |
| elasticsearch-type  | log    | no  |
| elasticsearch-username | no | no |  |
| elasticsearch-url   | no     | yes |
| elasticsearch-version | 5 | no |

###### elasticsearch-url ######

  - *url* to connect to the Elasticsearch cluster.
  - *examples*: http://127.0.0.1:9200, https://127.0.0.1:443

###### elasticsearch-insecure ######
  - *insecure* controls whether a client verifies the server's certificate chain and host name. If *insecure* is true, TLS accepts any certificate presented by the server and any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks.
  - *examples*: 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False

######  elasticsearch-index ######
  - *index* to write log messages to
  - *examples*: docker, logging

###### elasticsearch-username ######
  - *username* to authenticate to a secure Elasticsearch cluster
  - *example*: elastic

###### elasticsearch-password ######
  - *password* to authenticate to a secure Elasticsearch cluster
  - *WARNING*: the password will be stored as clear text password in the container config. This will be changed in the future versions.
  - *examples*: changeme

###### elasticsearch-type ######
  - *type* to write log messages to
  - *example*: log

###### elasticsearch-timeout ######
  - *timeout* maximum time in seconds that a connection is allowed to take
  - *example*: 10

###### elasticsearch-fields ######
  - *fields* to log to Elasticsearch Cluster
  - *examples*: containerID,containerLabels,containerEnv or none

###### elasticsearch-sniff ######

  - *sniff* uses the Node Info API to return the list of nodes in the cluster. It uses the list of URLs passed on startup plus the list of URLs found
 by the preceding sniffing process.
  - *examples*: 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False

###### elasticsearch-version ######
  - *version* of Elasticsearch cluster
  - *examples*: 1, 2, 5, 6

###### grok-pattern ######
  - *pattern* customer pattern
  - *examples*: CUSTOM_IP=(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)

###### grok-pattern-from ######
  - *pattern-from* add custom pattern from file or folder
  - *examples*: /srv/grok/pattern

###### grok-pattern-splitter ######
  - *pattern-splitter* is used for identifying multiple patterns from grok-pattern
  - *examples*: AND

###### grok-match ######
  - *match* the line to parse
  - *examples*: %{WORD:test1} %{WORD:test2}

###### grok-named-capture ######
  - *named-capture* parse all or only named captures
  - *examples*: 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False

### How to test ###

Creating and running a container:

```bash
$ docker run --rm -ti \
    --log-driver elasticsearch \
    --log-opt elasticsearch-url=https://127.0.0.1:9200 \
    --log-opt elasticsearch-insecure=false \
    --log-opt elasticsearch-username=elastic \
    --log-opt elasticsearch-password=changeme \
    --log-opt elasticsearch-sniff=false \
    --log-opt elasticsearch-index=docker \
    --log-opt elasticsearch-type=log \
    --log-opt elasticsearch-timeout=10 \
    --log-opt elasticsearch-version=5 \
    --log-opt elasticsearch-fields=containerID,containerName,containerImageID,containerImageName,containerCreated \
        alpine echo this is a test logging message
```

Search in Elasticsearch for the log message:

```bash
    $ curl 127.0.0.1:9200/docker/log/_search\?pretty=true

    {
        "_index" : "docker",
        "_type" : "log",
        "_id" : "AWCywmj6Dipxk6-_e8T5",
        "_score" : 1.0,
        "_source" : {
          "containerID" : "f7d986496f66",
          "containerName" => "focused_lumiere",
          "containerImageID" : "sha256:8d254d3d0dca3e3ee8f377e752af11e0909b51133da614af4b30e4769aff5a44",
          "containerImageName" : "alpine",
          "containerCreated" : "2018-01-18T21:45:29.053364087Z",
          "source" : "stdout",
          "timestamp" : "2018-01-18T21:45:30.294363869Z",
          "partial" : false
          "message" : "this is a test message\r"
        }
      }
```

### Description of fields

**Static Fields** are always present

| Field | Description | Default |
| ----- | ----------- | ------- |
| message  | The log message itself| yes |
| source | Source of the log message as reported by docker | yes |
| timestamp | Timestamp that the log was collected by the log driver | yes |
| partial | Whether docker reported that the log message was only partially collected | yes |

**Dynamic Fields**: can be provided by `elasticsearch-fields` log paramenter

| Field | Description | Default |
| ----- | ----------- | ------- |
| config | Config provided by log-opt | no |
| containerID | Id of the container that generated the log message | yes |
| containerName | Name of the container that generated the log message | yes |
| containerArgs | Arguments of the container entrypoint | no |
| containerImageID | ID of the container's image | no |
| containerImageName | Name of the container's image | yes |
| containerCreated | Timestamp of the container's creation | yes |
| containerEnv | Environment of the container | no |
| containerLabels | Label of the container | no |
| containerLogPath | Path of the container's Log | no |
| daemonName | Name of the container's daemon | no |
