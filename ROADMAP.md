# Roadmap

An overview of what will be done over the next few months.

## Docker Log Elasticsearch 1.5.1

Goals:

 * Write unit tests for an easier code review and better code quality.
 * Create a Continuous Integration of this project in order to avoid lots of manual interventions.
 * Captch labels and environments
 * Add an extra user option, e.g. `--log-opt elasticsearch-fields=containerName,containerID,containerLogLine` so for a free pick of docker info log
 * Strip ANSI colors
 * Add the capability of multilines for Java Exceptions
 * Add HTTPS Support and Skip Certificate Verify
 * Buffer logs into a file, if elasticsearch crashes. Add buffer size as well.
 * Implement bulk inserts
 * Implement queue size and batch size.
    * if queue is full, then write to file
    * if file buffer is full, discard messages
 * Implement Readlog capability
*  Implement grok for parsing docker logs
 * Add CONTRIBUTING file