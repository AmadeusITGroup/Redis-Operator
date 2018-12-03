#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -x

tmp=`mktemp`
echo '{"insecure-registries":["172.30.0.0/16"]}' > ${tmp}
sudo mv ${tmp} /etc/docker/daemon.json
sudo mount --make-shared /
sudo service docker restart
docker info
curl -Lo /tmp/oc.tar.gz https://github.com/openshift/origin/releases/download/v3.9.0/openshift-origin-client-tools-v3.9.0-191fece-linux-64bit.tar.gz && tar -xvf /tmp/oc.tar.gz -C /tmp && sudo mv /tmp/openshift-origin-client-tools-v3.9.0-191fece-linux-64bit/oc /usr/local/bin/oc
oc version
oc cluster up --version v3.9.0 --image openshift/origin --public-hostname 127.0.0.1 --routing-suffix apps.127.0.0.1.nip.io --host-data-dir ~/.oc/profiles/default/data --host-config-dir ~/.oc/profiles/default/config --host-pv-dir ~/.oc/profiles/default/pv --use-existing-config -e TZ=CET
oc login -u system:admin
oc adm policy --as system:admin add-cluster-role-to-user cluster-admin admin
oc project default
sudo ln -s /usr/local/bin/oc /usr/local/bin/ctl