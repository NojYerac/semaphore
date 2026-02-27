#! /usr/bin/env bash
set -e

trap 'catch $?' EXIT

catch () {
    if [[ "$1" != "0" ]]; then
        echo "Failed"
        exit $1
    fi
}

source $(dirname $0)/generate.sh

go build -ldflags "-X version.gitSHA=$(git rev-list -1 HEAD)" -o ./build/semaphore ./semaphore/main.go 
echo "Built semaphore binary"