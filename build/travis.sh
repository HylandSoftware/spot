#!/usr/bin/env bash
set -euo pipefail

SCRIPT_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

make
"${GOPATH}/bin/goveralls" -coverprofile="${SCRIPT_ROOT}/../coverage.out" -service=travis-ci

CGO_ENABLED=0 make build-unix
docker build "${SCRIPT_ROOT}/../" -f "${SCRIPT_ROOT}/Dockerfile" -t hylandsoftware/spot
