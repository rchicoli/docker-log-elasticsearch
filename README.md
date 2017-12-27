# Docker Log Elasticsearch

`docker-log-elasticsearch` forwards container logs to Elasticsearch service. Each log message will be written to a Elasticsearch service.

## Getting Started

You need to install Docker Engine >= 1.12 and Elasticsearch 5

Additional information about Docker plugins [can be found here].(https://docs.docker.com/engine/extend/plugins_logging/)

### Installing

To install the plugin, run

    docker plugin install rchicoli/docker-log-elasticsearch:latest --alias log2elasticsearch

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
        --log-driver docker-log-elasticsearch \
        --log-opt elasticsearch-address=http://127.0.0.1:9200 \
        --log-opt elasticsearch-index=docker \
        --log-opt elasticsearch-type=log \
            alpine echo this is a test logging message

## Output Format

Query elasticsearch:

    $ curl 127.0.0.1:9200/docker/_search\?pretty=true
    {
        "_index" : "docker",
        "_type" : "log",
        "_id" : "AWCXVahqU5saDepmAJRy",
        "_score" : 1.0,
        "_source" : {
            "Source" : "stdout",
            "@timestamp" : "2017-12-27T09:37:41.474630116Z",
            "partial" : false,
            "Config" : {
                "elasticsearch-address" : "http://127.0.0.1:9200",
                "elasticsearch-index" : "docker",
                "elasticsearch-type" : "log"
            },
            "ContainerID" : "dd14d704d73d5273ea8e9b4150ce1f1123875b0d7984644413ea8f3cf01b0718",
            "ContainerName" : "/ecstatic_goldwasser",
            "ContainerEntrypoint" : "echo",
            "ContainerArgs" : [
                "this",
                "is",
                "a",
                "logging",
                "message"
            ],
            "ContainerImageID" : "sha256:7328f6f8b41890597575cbaadc884e7386ae0acc53b747401ebce5cf0d624560",
            "ContainerImageName" : "alpine",
            "ContainerCreated" : "2017-12-27T09:37:40.637133648Z",
            "ContainerEnv" : [
                "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
            ],
            "ContainerLabels" : { },
            "LogPath" : "",
            "DaemonName" : "docker",
            "logline" : "this is a test logging message"
        }
    }

**Fields**

| Field | Description |
| ----- | ----------- |
| logline  | The log message itself|
| Source | Source of the log message as reported by docker |
| @Timestamp | Timestamp that the log was collected by the log driver |
| Partial | Whether docker reported that the log message was only partially collected |
| ContainerName | Name of the container that generated the log message |
| ContainerID | Id of the container that generated the log message |
| ContainerImageName | Name of the container's image |
| ContainerImageID | ID of the container's image |
| Err | Usually null, otherwise will be a string containing and error from the logdriver |