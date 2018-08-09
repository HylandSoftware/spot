#!/usr/bin/env bash
set -euo pipefail

SCRIPT_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

make
"${GOPATH}/bin/goveralls" -coverprofile="${SCRIPT_ROOT}/../coverage.out" -service=travis-ci

CGO_ENABLED=0 make build-unix

IMAGE_TAG=ci
if [ "${TRAVIS_PULL_REQUEST}" = "false" ]; then
    IMAGE_TAG=$(docker run -it --rm -v "$(pwd):/repo" gittools/gitversion /showvariable NuGetVersionV2 | tee /dev/tty)
fi

docker build "${SCRIPT_ROOT}/../" -f "${SCRIPT_ROOT}/../Dockerfile" -t "hylandsoftware/spot:${IMAGE_TAG%$'\r'}"

if [ "${TRAVIS_PULL_REQUEST}" = "false" ] && [ "${TRAVIS_BRANCH}" = "master" ]; then
    docker login -u "${DOCKER_USERNAME}" -p "${DOCKER_PASSWORD}"

    docker tag "hylandsoftware/spot:${IMAGE_TAG%$'\r'}" "hylandsoftware/spot:latest"
    docker push "hylandsoftware/spot:${IMAGE_TAG%$'\r'}"
    docker push "hylandsoftware/spot:latest"
fi