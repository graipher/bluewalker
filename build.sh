#!/usr/bin/env bash -eu

# Build bluewalker binary
# This script should be run on the same directory where Dockerfile is
# The architecture to build for can be given as parameter and will be
# given to go build in GOARCH enviroment variable.
# Built binary is copied to current directory

DEFAULT_GOARCH="amd64"

ARCH=${DEFAULT_GOARCH}

if [[ $# -gt 0 ]]; then
    ARCH=$1
fi

echo "Building bluewalker for ${ARCH}"

docker build --build-arg ARCH=${ARCH} -t bluewalker:build .
docker run  --rm -v $(pwd):/out bluewalker:build

echo "Build complete:"
ls -l ./bluewalker
