#!/bin/sh
set -e
set -x

docker-info \
    --loglevel=$LOG_LEVEL \
    --cache-timeout=$CACHE_TIMEOUT


