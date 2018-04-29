#!/usr/bin/env bash

set -e
echo "" > coverage.txt

for d in $(go list -f '{{if .TestGoFiles}}{{ .ImportPath }}{{end}}' ./... | grep -v vendor | grep -v test/e2e); do
    go test -race -coverprofile=profile.out -covermode=atomic $d
    if [ -f profile.out ]; then
        cat profile.out | grep -v zz_generated >> coverage.txt
        rm profile.out
    fi
done
