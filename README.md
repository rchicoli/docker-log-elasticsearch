# Docker Log Elasticsearch

[![GoDoc][1]][2] [![License: Apache 2.0][3]][4] [![Release][5]][6] [![Travis Status][7]][8] [![Coveralls Coverage][9]][10] [![Go Report Card][11]][12]

[1]: https://godoc.org/github.com/rchicoli/docker-log-elasticsearch?status.svg
[2]: https://godoc.org/github.com/rchicoli/docker-log-elasticsearch
[3]: https://img.shields.io/badge/License-Apache%202.0-blue.svg
[4]: LICENSE.md
[5]: https://img.shields.io/github/release/rchicoli/docker-log-elasticsearch.svg
[6]: https://github.com/rchicoli/docker-log-elasticsearch/releases/latest
[7]: https://travis-ci.org/rchicoli/docker-log-elasticsearch.svg?branch=master
[8]: https://travis-ci.org/rchicoli/docker-log-elasticsearch
[9]: https://coveralls.io/repos/rchicoli/docker-log-elasticsearch/badge.svg?branch=master&service=github
[10]: https://coveralls.io/github/rchicoli/docker-log-elasticsearch?branch=master
[11]: https://goreportcard.com/badge/github.com/rchicoli/docker-log-elasticsearch
[12]: https://goreportcard.com/report/github.com/rchicoli/docker-log-elasticsearch


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

### How to configure

The plugin must be disabled in order to change the [plugin settings](https://docs.docker.com/engine/reference/commandline/plugin_set/)

| Environment | Description | Default Value |
| ----- | ----------- | -------------- |
| LOG_LEVEL | log level to output for plugin logs (debug, info, warn, error) | info |
| TZ        | time zone to generate new indexes at midnight | none |

### How to use

#### Prerequisites

Before creating a docker container, a healthy instance of Elasticsearch service must be running.

##### Options #####

| Key | Default Value | Required |
| --- | ------------- | -------- |
| elasticsearch-fields | containerID,containerName,containerImageName,containerCreated | no |
| elasticsearch-index | docker-%Y.%m.%d | no  |
| elasticsearch-insecure | false | no |
| elasticsearch-password | no | no |  |
| elasticsearch-sniff | yes | no | |
| elasticsearch-timeout | 1    | no  |
| elasticsearch-type  | log    | no  |
| elasticsearch-username | no | no |  |
| elasticsearch-url   | no     | yes |
| elasticsearch-version | 5 | no |
| elasticsearch-bulk-actions | 100 | no |
| elasticsearch-bulk-size | 5242880 | no |
| elasticsearch-bulk-flush-interval | 5s | no |
| elasticsearch-bulk-workers | 1 | no |
| grok-named-capture | true | no |
| grok-pattern | no | no |
| grok-pattern-from | no | no |
| grok-pattern-splitter |  and  | no |
| grok-pattern-match | no | no |

###### elasticsearch-url ######

  - *url* to connect to the Elasticsearch cluster.
  - *examples*: http://127.0.0.1:9200, https://127.0.0.1:443

###### elasticsearch-insecure ######
  - *insecure* controls whether a client verifies the server's certificate chain and host name. If *insecure* is true, TLS accepts any certificate presented by the server and any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks.
  - *examples*: 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False

######  elasticsearch-index ######
  - *index* to write log messages to
  - *examples*: docker, logging-%F, docker-%Y.%m.%d

Interpreted sequences are:

| Regex | Description |
| ----- | ----------- |
| %b    | locale's abbreviated month name (Jan) |
| %B    | locale's full month name (January) |
| %d    | day of month (01) |
| %F    | full date; same as %Y.%m.%d |
| %j    | day of year (001..366) |
| %m    | month (01..12) |
| %y    | last two digits of year (00..99) |
| %Y    | year (2018) |

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

###### elasticsearch-bulk-actions ######
  - *bulk-actions* specifies when to flush based on the number of actions currently added
  - *examples*: 1000

###### elasticsearch-bulk-workers ######
  - *bulk-workers* is the number of concurrent workers allowed to be executed
  - *examples*: 2

###### elasticsearch-bulk-size ######
  - *bulk-size* specifies when to flush based on the size (in bytes) of the actions currently added. Set to -1 to disable it.
  - *examples*: 1024, -1

###### elasticsearch-bulk-flush-interval ######
  - *bulk-flush-interval* specifies when to flush at the end of the given interval
  - *examples*: 300ms, 1s, 2h45m

###### grok-pattern ######
  - *pattern* add customer pattern
  - *examples*: CUSTOM_IP=(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)

###### grok-pattern-from ######
  - *pattern-from* add custom pattern from a file or folder
  - *examples*: /srv/grok/pattern (this directory must be bound or linked inside the plugins's rootfs)

###### grok-pattern-splitter ######
  - *pattern-splitter* is used for splitting multiple patterns from grok-pattern
  - *examples*: " AND " (with white spaces before and after the word AND)

###### grok-match ######
  - *match* the log line to parse
  - *examples*: %{WORD:test1} %{WORD:test2}

###### grok-named-capture ######
  - *named-capture* parse each inner pattern or only named captures
  - *examples*: 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False

### How to test ###

1 - Creating and running a container:

```bash
$ docker run --rm -ti \
    --log-driver elasticsearch \
    --log-opt elasticsearch-url=https://127.0.0.1:9200 \
    --log-opt elasticsearch-insecure=false \
    --log-opt elasticsearch-username=elastic \
    --log-opt elasticsearch-password=changeme \
    --log-opt elasticsearch-sniff=false \
    --log-opt elasticsearch-index=docker-%F \
    --log-opt elasticsearch-type=log \
    --log-opt elasticsearch-timeout=10 \
    --log-opt elasticsearch-version=5 \
    --log-opt elasticsearch-fields=containerID,containerName,containerImageID,containerImageName,containerCreated \
    --log-opt elasticsearch-bulk-workers=1 \
    --log-opt elasticsearch-bulk-actions=1000 \
    --log-opt elasticsearch-bulk-size=1024 \
    --log-opt elasticsearch-bulk-flush-interval=1s \
    --log-opt elasticsearch-bulk-stats=false \
        alpine echo -n "this is a test logging message"
```

Search in Elasticsearch for the log message:

```bash
$ curl 127.0.0.1:9200/docker/log/_search\?pretty=true

{
  "_index" : "docker-2006.01.02",
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
    "partial" : false,
    "message" : "this is a test message"
  }
}
```

2 - Using grok extension for parsing the log messages:

```bash
docker run --rm -ti \
    --log-driver rchicoli/docker-log-elasticsearch:latest \
    --log-opt elasticsearch-url=https://127.0.0.1:9200 \
    --log-opt grok-match='%{COMMONAPACHELOG}' \
    --log-opt grok-named-capture=true \
    alpine echo -n "127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] \"GET /index.php HTTP/1.1\" 404 $((RANDOM))"
```

The apache log line above will be displayed in elasticsearch as following:

```bash
"_source": {
  "containerID": "af7b2f782963",
  "containerName": "dazzling_knuth",
  "containerImageName": "alpine",
  "containerCreated": "2018-03-16T22:13:42.27376049Z",
  "source": "stdout",
  "timestamp": "2018-03-16T22:13:42.810856646Z",
  "partial": true,
  "grok": {
    "auth": "-",
    "bytes": "28453",
    "clientip": "127.0.0.1",
    "httpversion": "1.1",
    "ident": "-",
    "rawrequest": "",
    "request": "/index.php",
    "response": "404",
    "timestamp": "23/Apr/2014:22:58:32 +0200",
    "verb": "GET"
  }
}
```

In case you want to save the complete log line and its meta fields, you can set `grok-named-capture` to `false`, this will show some duplicated values though:

```bash
"grok": {
  "BASE10NUM": "16406",
  "COMMONAPACHELOG": "127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] \"GET /index.php HTTP/1.1\" 404 16406",
  "EMAILADDRESS": "",
  "EMAILLOCALPART": "",
  "HOSTNAME": "",
  "HOUR": "22",
  "INT": "+0200",
  "IP": "127.0.0.1",
  "IPV4": "127.0.0.1",
  "IPV6": "",
  "MINUTE": "58",
  "MONTH": "Apr",
  "MONTHDAY": "23",
  "SECOND": "32",
  "TIME": "22:58:32",
  "USER": "-",
  "USERNAME": "-",
  "YEAR": "2014",
  "auth": "-",
  "bytes": "16406",
  "clientip": "127.0.0.1",
  "httpversion": "1.1",
  "ident": "-",
  "rawrequest": "",
  "request": "/index.php",
  "response": "404",
  "timestamp": "23/Apr/2014:22:58:32 +0200",
  "verb": "GET"
}
```

3 - There are two different ways of adding custom grok patterns.

a. by providing the grok pattern as parameter, e.g.:

```bash
docker run --rm -ti --log-opt grok-named-capture=false \
  --log-driver rchicoli/docker-log-elasticsearch:development \
  --log-opt elasticsearch-url=http://172.31.0.2:9200 \
  --log-opt grok-pattern='MY_NUMBER=(?:[+-]?(?:[0-9]+)) && MY_USER=[a-zA-Z0-9._-]+ && MY_PATTERN=%{MY_NUMBER:random_number} %{MY_USER:user}' \
  --log-opt grok-pattern-splitter=' && ' \
  --log-opt grok-match='%{MY_PATTERN:log}' \
  alpine echo -n "$((RANDOM)) tester"
```

Note we are using multiple custom patterns and also a different pattern splitter.

```bash
"_source": {
  "containerID": "e11b45c911b4",
  "containerName": "ecstatic_leakey",
  "containerImageName": "alpine",
  "containerCreated": "2018-03-16T22:47:39.39245724Z",
  "source": "stdout",
  "timestamp": "2018-03-16T22:47:39.940595233Z",
  "partial": true,
  "grok": {
    "log": "2921 tester",
    "random_number": "2921",
    "user": "tester"
  }
}
```

b. by providing a directory with different grok patterns in it or just a single file, e.g.:

At first, you have to place the file or directory inside the docker's rootfs. It is up to you to choose the right way to do it.
You could link the file, mount the directory or simply copy it inside `/var/lib/docker/plugin/<plugin-id>/rootfs/`. Afterwards you can pass it as following:

```bash
docker run -ti --rm --log-driver rchicoli/docker-log-elasticsearch:development --log-opt elasticsearch-url=http://172.31.0.2:9200 \
  --log-opt grok-pattern-from="/patterns" \
  rchicoli/webapper
```

4 - If grok is not able to parse the log line, then it will still send the unparsed message to Elasticsearch with an error description, e.g.:

```bash
"grok": {
  "err": "grok pattern does not match line",
  "line": "4721 tester"
}
```

### Limitations

There are some limitations so far, which will be improved at some point.

  - grok parses everything to a string field, convertion type is only possible if a template has been added to elasticsearch
  - grok-pattern-from requires the file to be inside the plugin's rootfs. Alternative is the plugin's mount source.

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
