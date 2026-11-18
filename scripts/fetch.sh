#!/bin/sh
# Clone pinned upstream source without changing existing source checkouts.
set -eu
repo=$1
revision=$2
case "$repo" in openvue|openicons) ;; *) exit 2 ;; esac
target=".cache/$repo"
if [ -e "$target" ]; then
    actual=$(git -C "$target" rev-parse HEAD)
    if [ "$actual" != "$revision" ]; then
        echo "$target is at $actual, expected $revision. Move it aside before fetching." >&2
        exit 1
    fi
else
    mkdir -p .cache
    git init "$target"
    git -C "$target" remote add origin "https://github.com/openvi-foundation/$repo.git"
    git -C "$target" fetch --depth 1 origin "$revision"
    git -C "$target" checkout --detach FETCH_HEAD
fi
if [ -n "$(git -C "$target" status --porcelain)" ]; then
    echo "$target has local changes. Use a clean checkout for reproducible docsets." >&2
    exit 1
fi
