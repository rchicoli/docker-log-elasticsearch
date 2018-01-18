#!/bin/bash

base_dir=$(dirname `readlink -f "$0"`)
docker_compose_file="${base_dir}/../docker/docker-compose.yml"
makefile="${base_dir}/../Makefile"

exit_code=0

# compile and install docker plugin
if sudo BASE_DIR="$base_dir/.." make -f "$makefile"; then

    # elasticsearch bootstrap
    # max virtual memory areas vm.max_map_count [65530] is too low, increase to at least [262144]
    sudo sysctl -q -w vm.max_map_count=262144

    # create and run elasticsearch as a container
    if docker-compose -f "$docker_compose_file" up -d elasticsearch; then

        # create a container for logging to elasticsearch
        if "${base_dir}/./wait-for.sh" elasticsearch 9200 docker-compose -f "$docker_compose_file" up -d webapper; then

            # create some tests tests
            sample_message="this-is-one-logging-line"
            curl "http://172.31.0.3:8080/$sample_message" &>/dev/null

            # wait couple of seconds for the message to be processed by elasticsearch
            sleep 3

            if curl -q http://172.31.0.2:9200/docker-compose/ci/_search\?pretty=true | grep "$sample_message"; then
                echo "it works like a charm"
            else
                echo "something went wrong"
                exit_code=1
            fi

        else
            exit_code=1
        fi

    else
        exit_code=1
    fi

else
    exit 1
fi

# post tasks
docker-compose -f "$docker_compose_file" rm --stop --force

if [ $exit_code -ne 0 ]; then
    exit 1
fi