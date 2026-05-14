#!/bin/sh

URL=$1

echo "Waiting for $URL..."

for i in $(seq 1 10); do
  curl -k --fail "$URL" && exit 0
  sleep 1
done

echo "Service not ready"
exit 1