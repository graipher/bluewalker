# Build bluewalker binary with latest golang
# Builds the binary when the image is built and when it is run copies
# the built binary into directory /out
#
# Use build.sh script to build and run the docker image
FROM golang:latest

ARG ARCH="amd64"

WORKDIR /build/bluewalker/
COPY . .
RUN GOOS="linux" GOARCH=${ARCH} go build
CMD cp /build/bluewalker/bluewalker /out/
