#!/bin/bash

GIT_ROOT=$(git rev-parse --show-toplevel)
export GOPATH=$GIT_ROOT/../../../../

oc cluster up
echo "login to openshift as admin"
oc login -u system:admin --insecure-skip-tls-verify=true
oc adm policy --as system:admin add-cluster-role-to-user cluster-admin admin

echo "create the helm tiller"
oc new-project tiller
export TILLER_NAMESPACE=tiller
oc project tiller
oc process -f ${GIT_ROOT}/hack/oc-tiller-template.yaml -p TILLER_NAMESPACE="${TILLER_NAMESPACE}" | oc create -f -
oc policy add-role-to-user edit "system:serviceaccount:${TILLER_NAMESPACE}:tiller"
oc create clusterrolebinding tiller-cluster-rule --clusterrole=cluster-admin --serviceaccount="$TILLER_NAMESPACE":tiller
helm init --client-only

# -----
printf "Waiting for tiller deployment to complete."
until [ $(oc get deployment -n tiller tiller -ojsonpath="{.status.conditions[?(@.type==\"Available\")].status}") == "True" ] > /dev/null 2>&1; do sleep 1; printf "."; done
echo

echo "Install the redis-cluster operator"
oc project default
oc create clusterrolebinding redis-operators --clusterrole=cluster-admin --serviceaccount=default:redis-operator
echo "First build the container"
make TAG=latest container
# tag the same image for rolling-update test
docker tag redisoperator/redisnode:latest redisoperator/redisnode:4.0

echo "create RBAC for rediscluster"
oc create -f $GIT_ROOT/examples/RedisCluster_RBAC.yaml

printf  "create and install the redis operator in a dedicate namespace"
until helm install --namespace default -n operator chart/redis-operator; do sleep 1; printf "."; done
echo

printf "Waiting for redis-operator deployment to complete."
until [ $(oc get deployment operator-redis-operator -ojsonpath="{.status.conditions[?(@.type==\"Available\")].status}") == "True" ] > /dev/null 2>&1; do sleep 1; printf "."; done
echo

echo "[[[ Run End2end test ]]] "
cd ./test/e2e && go test -c && ./e2e.test --kubeconfig=$HOME/.kube/config --ginkgo.slowSpecThreshold 260

echo "[[[ Cleaning ]]]"

echo "wait the garbage collection"
sleep 20 
echo "Remove redis-operator helm chart"
helm del --purge operator

oc delete ClusterRole redis-operator, redis-node
oc delete ServiceAccount redis-operator, redis-node

oc cluster down
