#!/bin/bash

status=$(docker inspect --format='{{json .State.Health.Status}}' testkit-es)

# shellcheck disable=SC2034
for i in {1..50} ; do
  if [ "$status" == "\"healthy\"" ]; then
    echo "Elasticsearch is healthy"
    exit 0
  fi
  echo "Waiting for Elasticsearch to be healthy"
  sleep 1
  status=$(docker inspect --format='{{json .State.Health.Status}}' testkit-es)
done