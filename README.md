# Docker Log Elasticsearch

`docker-log-elasticsearch` forwards container logs to Elasticsearch service.

This application is under active development and will continue to be modified and improved over time. The current release is an "alpha." (see [Roadmap](ROADMAP.md)).

## Releases

| Branch Name | Docker Tag | Elasticsearch Version | Remark |
| ----------- | ---------- | --------------------- | ------ |
| master      | 1.0.x      | 1.x, 2.x, 5.x, 6.x    | Future stable release. |
| alpha       | 0.0.1, 0.2.1   | 1.x, 2.x, 5.x, 6.x   | Actively alpha release. |

```
release-0.1.1
        | | |_ bug fixes
        | |___ new features
        |_____ release version
```

## Getting Started

You need to install Docker Engine >= 1.12 and Elasticsearch application

Additional information about Docker plugins [can be found here](https://docs.docker.com/engine/extend/plugins_logging/).

### Installing

To install the plugin, run

    docker plugin install rchicoli/docker-log-elasticsearch:0.1.1 --alias elasticsearch

This command will pull and enable the plugin

### Using

First of all, a healthy instance of Elasticsearch service must be running.

#### Note

To run a specific container with the logging driver:

    Use the --log-driver flag to specify the plugin.
    Use the --log-opt flag to specify the URL for the HTTP connection and further options.

**Options**

| Key | Default Value | Required | Examples |
| --- | ------------- | -------- | ------- |
| elasticsearch-url   | no     | yes | http://127.0.0.1:9200 |
| elasticsearch-index | docker | no  | docker-logs |
| elasticsearch-type  | log    | no  | docker-plugin |
| elasticsearch-timeout | 1    | no  | 10 |
| elasticsearch-fields | containerID,containerName,containerImageName,containerCreated | no | containerID,containerLabels,containerEnv |
| elasticsearch-version | 5 | no | 1, 2, 5, 6 |


#### Testing

Creating and running a container:

    $ docker run --rm  -ti \
        --log-driver elasticsearch \
        --log-opt elasticsearch-url=http://127.0.0.1:9200 \
        --log-opt elasticsearch-index=docker \
        --log-opt elasticsearch-type=log \
        --log-opt elasticsearch-timeout=10 \
        --log-opt elasticsearch-version=5 \
        --log-opt elasticsearch-fields=containerID,containerName,containerImageID,containerImageName,containerCreated \
            alpine echo this is a test logging message

## Output Format

Query elasticsearch:

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