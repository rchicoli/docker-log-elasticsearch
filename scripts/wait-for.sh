#!/bin/bash

set -e

host=${1:?"unknown destination host"}
port=${2:?"unknown port"}
shift 2

cmd=${@:?"missing command parameter"}

timeout=30
count=0
until ip=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $host 2>/dev/null); do

    if [ $count -lt $timeout ]; then
        count=$(($count+1));
    else
        echo "timing out: $host container has no ip address"
        exit 1
    fi

    sleep 1

done


timeout=30
count=0
until nc -z "$ip" "$port" >/dev/null; do

    if [ $count -lt $timeout ]; then
        count=$(($count+1));
    else
        echo "timing out: $host took too long to start up"
        exit 1
    fi

    sleep 1

done

exec $cmd
