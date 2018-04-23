#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

./vendor/k8s.io/code-generator/generate-groups.sh all github.com/amadeusitgroup/redis-operator/pkg/client github.com/amadeusitgroup/redis-operator/pkg/api redis:v1 --go-header-file ./hack/custom-boilerplate.go.txt
