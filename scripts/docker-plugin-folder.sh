#!/bin/bash

set -e
container_name=${1:?"unkown plugin name"}
resource_name=${2:?"unkown folder name"}

# folder_name=${resource_name%/*}
file_name=${resource_name##*/}

container_id="$(docker plugin ls | awk  -v name=$container_name '$NF ~ /true/ && $2 ~ name {print $1}')"

shopt -s extglob
plugin_folder=$(ls -1d /var/lib/docker/plugins/${container_id}*)

if ! test -d "$plugin_folder"; then
    echo "plugin folder not found"
    exit 1
fi

# share_folder="${plugin_folder}/rootfs/tmp/${folder_name}"

# if ! test -d "$share_folder" && [[ "$folder_name" != "$file_name" ]]; then
#     mkdir -pv "$share_folder"
# fi

cp -vr "$resource_name" "${plugin_folder}/rootfs/tmp/"
