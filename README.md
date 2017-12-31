# Docker Log Elasticsearch

`docker-log-elasticsearch` forwards container logs to Elasticsearch service.

This application is under active development and will continue to be modified and improved over time. The current release is an "alpha." (see [Roadmap](ROADMAP.md)).

## Releases

| Branch Name | Docker Tag | Elasticsearch Version | Remark |
| ----------- | ---------- | --------------------- | ------ |
| release-1.5.x  | 1.5.x   | 5.x                | Future stable release. |
| alpha-0.5.x    | 0.5.1, 0.5.2   | 5.x                | Actively alpha release. |

```
release-0.5.1
        | | |_ new features or bug fixes
        | |___ elasticsearch major version
        |_____ release version
```

## Getting Started

You need to install Docker Engine >= 1.12 and Elasticsearch 5

Additional information about Docker plugins [can be found here](https://docs.docker.com/engine/extend/plugins_logging/).

### Installing

To install the plugin, run

    docker plugin install rchicoli/docker-log-elasticsearch:0.5.1 --alias elasticsearch

This command will pull and enable the plugin

### Using

First of all, a healthy instance of Elasticsearch service must be running.

#### Note

To run a specific container with the logging driver:

    Use the --log-driver flag to specify the plugin.
    Use the --log-opt flag to specify the URL for the HTTP connection and further options.

#### Testing

Creating and running a container:

    $ docker run --rm  -ti \
        --log-driver elasticsearch \
        --log-opt elasticsearch-address=http://127.0.0.1:9200 \
        --log-opt elasticsearch-index=docker \
        --log-opt elasticsearch-type=log \
            alpine echo this is a test logging message

## Output Format

Query elasticsearch:

```bash
    $ curl 127.0.0.1:9200/docker/_search\?pretty=true

    {
        "_index" : "docker",
        "_type" : "log",
        "_id" : "AWCYGwapnY8fJx4hGldT",
        "_score" : 1.0,
        "_source" : {
          "source" : "stdout",
          "@timestamp" : "2017-12-27T13:13:16.182379456Z",
          "partial" : false,
          "config" : {
            "elasticsearch-address" : "http://127.0.0.1:9200",
            "elasticsearch-index" : "docker",
            "elasticsearch-type" : "log"
          },
          "containerID" : "0ed70784b72b7b40140d42e8aa69b30ecd12daa186942d5d8ee6341a7ef0c31e",
          "containerName" : "/festive_hawking",
          "containerEntrypoint" : "echo",
          "containerArgs" : [
            "this",
            "is",
            "a",
            "test",
            "logging",
            "message"
          ],
          "containerImageID" : "sha256:7328f6f8b41890597575cbaadc884e7386ae0acc53b747401ebce5cf0d624560",
          "containerImageName" : "alpine",
          "containerCreated" : "2017-12-27T13:13:15.384933884Z",
          "containerEnv" : [
            "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
          ],
          "containerLabels" : { },
          "logPath" : "",
          "daemonName" : "docker",
          "message" : "this is a test logging message\r"
        }
      }
```

**Fields**

| Field | Description |
| ----- | ----------- |
| message  | The log message itself|
| source | Source of the log message as reported by docker |
| @timestamp | Timestamp that the log was collected by the log driver |
| partial | Whether docker reported that the log message was only partially collected |
| containerName | Name of the container that generated the log message |
| containerID | Id of the container that generated the log message |
| containerImageName | Name of the container's image |
| containerImageID | ID of the container's image |
| containerLabels | Label of the container |
| err | Usually null, otherwise will be a string containing and error from the logdriver |