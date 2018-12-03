#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -x

export ROOT=$(dirname "${BASH_SOURCE}")/..
echo "TEST $CLUSTER"

printenv

ctl=/usr/local/bin/kubectl
if [ "$CLUSTER" == "openshift" ]; then
    echo "INIT Openshift test platform"
    ./hack/ci-openshift-install.sh
    ctl=/usr/local/bin/oc
else
    echo "INIT Kubernetes test platform"
    ./hack/ci-minikube-install.sh
fi

# "ctl" command is in fact "kubectl" or "oc" depending of the CLUSTER var env value
# common part
JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'; until $ctl get nodes -o jsonpath="$JSONPATH" 2>&1 | grep -q "Ready=True"; do sleep 1; done
$ctl get nodes
$ctl get pods --all-namespaces
curl https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get | bash
$ctl -n kube-system create sa tiller
$ctl create clusterrolebinding tiller --clusterrole cluster-admin --serviceaccount=kube-system:tiller
helm init --wait --service-account tiller

make -C $ROOT build
make -C $ROOT test
make -C $ROOT TAG=$TAG container
make -C $ROOT TAG=4.0 container
docker images
helm install --wait --version $TAG -n end2end-test --set image.pullPolicy=IfNotPresent --set image.tag=$TAG chart/redis-operator
while [ $($ctl get pod --selector=app=redis-operator --all-namespaces | grep 'redis-operator' | awk '{print $4}') != "Running" ]
do
   echo "$($ctl get pod --selector=app=redis-operator --all-namespaces)"
   sleep 5
done

$ctl logs -f $($ctl get pod -l app=redis-operator --output=jsonpath={.items[0].metadata.name}) > /tmp/tmp.operator.logs &

cd ./test/e2e
EXIT_CODE=0
go test -c && ./e2e.test --kubeconfig=$HOME/.kube/config --image-tag=$TAG --ginkgo.slowSpecThreshold 350 || EXIT_CODE=$? && true ;
echo $EXIT_CODE
helm delete end2end-test
cat /tmp/tmp.operator.logs
exit $EXIT_CODE
