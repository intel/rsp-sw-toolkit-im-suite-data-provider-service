#!/bin/bash
# data-provider-service

set -e

echo -e "Go Building"
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo

echo -e "Making docker image"
docker build -t data-provider-service .