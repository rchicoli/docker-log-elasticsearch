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

    docker plugin install rchicoli/docker-log-elasticsearch:0.5.6 --alias elasticsearch

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


#### Testing

Creating and running a container:

    $ docker run --rm  -ti \
        --log-driver elasticsearch \
        --log-opt elasticsearch-url=http://127.0.0.1:9200 \
        --log-opt elasticsearch-index=docker \
        --log-opt elasticsearch-type=log \
        --log-opt elasticsearch-timeout=10 \
            alpine echo this is a test logging message

## Output Format

Query elasticsearch:

```bash
    $ curl 127.0.0.1:9200/docker/_search\?pretty=true

    {
        "_index" : "docker",
        "_type" : "log",
        "_id" : "AWCywmj6Dipxk6-_e8T5",
        "_score" : 1.0,
        "_source" : {
          "source" : "stdout",
          "@timestamp" : "2018-01-01T17:26:13.489630708Z",
          "config" : null,
          "containerID" : "abfed8ebf755f16762550b6b0eaaf612b7051513e64de64db4c93ba4913d0c4f",
          "containerName" : "/amazing_bardeen",
          "containerImageName" : "alpine",
          "containerCreated" : "2018-01-01T17:26:12.622116023Z",
          "message" : "this is a test message\r"
        }
      }
```

**Fields**

| Field | Description | Default |
| ----- | ----------- | ------- |
| message  | The log message itself| yes |
| source | Source of the log message as reported by docker | yes |
| @timestamp | Timestamp that the log was collected by the log driver | yes |
| partial | Whether docker reported that the log message was only partially collected | no |
| containerID | Id of the container that generated the log message | no |
| containerName | Name of the container that generated the log message | yes |
| containerArgs | Arguments of the container entrypoint | no |
| containerImageID | ID of the container's image | no |
| containerImageName | Name of the container's image | yes |
| containerCreated | Timestamp of the container's creation | yes |
| containerEnv | Environment of the container | no |
| containerLabels | Label of the container | no |
| containerLogPath | Path of the container's Log | no |
| daemonName | Name of the container's daemon | no |
| err | Usually null, otherwise will be a string containing and error from the logdriver |