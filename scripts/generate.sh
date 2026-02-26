#! /usr/bin/env bash
set -e

trap 'catch $?' EXIT

catch () {
    if [[ "$1" != "0" ]]; then
        echo "Failed"
        exit $1
    fi
}

project_root=$(cd $(dirname $0)/.. && pwd)

protoc -I ${project_root}/api  ${project_root}/api/*.proto --go_out=${project_root} --go-grpc_out=${project_root}
echo "Generated gRPC code"
