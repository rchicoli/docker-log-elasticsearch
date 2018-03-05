# Roadmap

An overview of what will be done over the next few months.

## Docker Log Elasticsearch 2.0.0

Goals:

 - [ ] Strip ANSI colors
 - [ ] Create an API for dumping or changing config on the fly
 - [ ] Parse partial log messages and merge them, if wished
 - [ ] Add performance tests

## Docker Log Elasticsearch 1.0.0

Goals:

 - [ ] Write unit tests
 - [X] Create a Continuous Integration of this project in order to avoid lots of manual interventions.
 - [ ] Captch labels and environments
 - [X] Add an extra user option, e.g. `--log-opt elasticsearch-fields=containerName,containerID,containerLogLine` so for a free pick of docker info log
 - [ ] Add the capability of multilines for Java Exceptions
 - [X] Add HTTPS Support and Skip Certificate Verify
 - [ ] Buffer logs into a file, if elasticsearch crashes. Add buffer size as well.
 - [ ] Implement bulk inserts
 - [ ] Implement queue size and batch size
   - [ ] if queue is full, then write to file
   - [ ] if file buffer is full, discard messages
 - [ ] Implement Readlog capability
 - [ ] Implement grok for parsing docker logs
 - [ ] Add CONTRIBUTING file
