#!/bin/bash

unset GOROOT

make build-${OS} ARCH=${ARCH} RELEASE_VERSION="${RELEASE_VERSION}"
