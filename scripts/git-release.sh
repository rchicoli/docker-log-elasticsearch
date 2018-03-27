#!/bin/bash

set -e

# fetch all tags from the remote
git fetch --tags

# find last tag
LAST_TAG="$(git describe --tags "$(git rev-list --tags --max-count=1)" 2>/dev/null)"

# create a new tag version
if [[ -n "${RELEASE_TAG:+x}" ]]; then
    NEW_TAG="$RELEASE_TAG"
elif [[ -n "${MAJOR_RELEASE:+x}" ]]; then
    NEW_TAG="$(awk -F. '{printf "%d.0.0", $1+1 }' <(echo "$LAST_TAG"))"
elif [[ -n "${FEATURE_RELEASE:+x}" ]]; then
    NEW_TAG="$(awk -F. '{printf "%d.%d.0", $1, $2+1 }' <(echo "$LAST_TAG"))"
else
    NEW_TAG="$(awk -F. '{printf "%d.%d.%d", $1, $2, $3+1}' <(echo "$LAST_TAG"))"
fi

if [[ "$NEW_TAG" == "$LAST_TAG" ]]; then
    NEW_TAG="$(awk -F. '{printf "%d.%d.%d", $1, $2, $3+1}' <(echo "$NEW_TAG"))"
fi

export PLUGIN_TAG="$NEW_TAG"
make

docker login -u "$DOCKER_USERNAME" -p "$DOCKER_PASSWORD" &>/dev/null
make push

# git log --oneline "${LAST_TAG}..HEAD"

# FIXME: travis deploy extension cuts multiple commit lines
# possible solution: remove deploy extension and add git push tag command
git tag -a "$NEW_TAG" -m "$(git log --oneline ${LAST_TAG}..HEAD)"

git tag -n100
