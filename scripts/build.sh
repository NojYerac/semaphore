#! /usr/bin/env bash

go build -ldflags "-X version.gitSHA=$(git rev-list -1 HEAD)" -o ./build/semaphore ./semaphore/main.go 