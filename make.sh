#!/bin/sh
set -eo pipefail

tgt=RethinkRAW.app/Contents/MacOS

if [[ "$1" == test ]]; then
    echo Test build...
    go build -o "$tgt/rethinkraw"
    exec "$tgt/rethinkraw"
else
    echo Release build...
    go clean
    go generate
    go build -tags memfs -ldflags "-s -w" -o "$tgt/rethinkraw"
    go mod tidy
    rm -rf "$tgt/data"
fi