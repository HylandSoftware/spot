#!/usr/bin/env bash
set -euo pipefail

SCRIPT_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

make restore
make test

"${GOPATH}/bin/goveralls" -coverprofile="${SCRIPT_ROOT}/../coverage.out" -service=travis-ci