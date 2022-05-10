#!/bin/bash

unset GOROOT

make build-${OS} ARCH=${ARCH} RELEASE_VERSION="${RELEASE_VERSION}"

NAME=pull-request-protector-buildkite-plugin
RELEASE_VERSION?= "0.0.0"
ARCH="${ARCH-amd64}"
COMMIT=$(shell git rev-parse --short=7 HEAD)
TIMESTAMP:=$(shell date -u '+%Y-%m-%dT%I:%M:%SZ')

# set linker flags
LDFLAGS += -X main.BuildTime=${TIMESTAMP}
LDFLAGS += -X main.BuildSHA=${COMMIT}
LDFLAGS += -X main.Version=${RELEASE_VERSION}

# clean old output
rm -rf ${NAME}-$*-${ARCH}

.PHONY: build
build-%: clean-%
	GOOS=$* GOARCH=${ARCH} CGO_ENABLED=0 go build -ldflags '${LDFLAGS}' -o ${PWD}/${NAME}-$*-${ARCH}
