#!/bin/bash

if [ -z "$1" ]; then
    echo "please provide the version as parameter: ./helm-release.sh <version>  <chart-name>\n"
    exit 1
fi

if [ -z "$2" ]; then
    echo "please provide the chart name as parameter: ./helm-release.sh <version> <chart-name>\n"
    exit 1
fi

cd $(git rev-parse --show-toplevel)
helm package --version "$1" chart/$2
mv "helm-$2-$1.tgz" docs/
helm repo index docs --url https://amadeusitgroup.github.io/redis-operator/ --merge docs/index.yaml
git add --all docs/
