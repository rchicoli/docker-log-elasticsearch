#!/bin/bash

# fetch all tags from the remote
git fetch --tags

# find last tag
if LAST_TAG="$(git describe --tags "$(git rev-list --tags --max-count=1)" 2>/dev/null)"; then
    # create a new tag version
    NEW_TAG="$(awk -F. '{printf "%d.%d.%d", $1, $2, $3+1}' <(echo "$LAST_TAG"))"
else
    exit 1
fi

git tag -a "$NEW_TAG" -m "new release"

export PLUGIN_TAG="$NEW_TAG"

if make; then
    if ! make push; then
        echo "error: make push failed"
        exit 1
    fi
else
    echo "error: make failed"
    exit 1
fi

# publish new tag
git push -u origin "$NEW_TAG"