#!/bin/bash

GIT_ROOT=$(git rev-parse --show-toplevel)
export GOPATH=$GIT_ROOT/../../../../

echo "Start minikube with RBAC option"
minikube start --extra-config=apiserver.Authorization.Mode=RBAC

echo "Create the missing rolebinding for k8s dashboard"
kubectl create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default

echo "Create the cluster role binding for the helm tiller"
kubectl create clusterrolebinding tiller-cluster-admin  --clusterrole=cluster-admin --serviceaccount=kube-system:default

echo "Init the helm tiller"
helm init

printf "Waiting for tiller deployment to complete."
until [ $(kubectl get deployment -n kube-system tiller-deploy -ojsonpath="{.status.conditions[?(@.type=='Available')].status}") == "True" ] > /dev/null 2>&1; do sleep 1; printf "."; done
echo

eval $(minikube docker-env)
echo "Install the redis-cluster operator"

echo "First build the container"
make TAG=latest container
# tag the same image for rolling-update test
docker tag redis-operator/redisnode:latest redis-operator/redisnode:4.0

echo "create RBAC for rediscluster"
kubectl create -f $GIT_ROOT/examples/RedisCluster_RBAC.yaml

printf  "create and install the redis operator in a dedicate namespace"
until helm install -n operator chart/redis-operator; do sleep 1; printf "."; done
echo

printf "Waiting for redis-operator deployment to complete."
until [ $(kubectl get deployment operator-redis-operator -ojsonpath="{.status.conditions[?(@.type=='Available')].status}") == "True" ] > /dev/null 2>&1; do sleep 1; printf "."; done
echo

echo "[[[ Run End2end test ]]] "
cd ./test/e2e && go test -c && ./e2e.test --kubeconfig=$HOME/.kube/config --ginkgo.slowSpecThreshold 260

echo "[[[ Cleaning ]]]"

echo "Remove redis-operator helm chart"
helm del --purge operator

kubectl delete ClusterRole redis-operator, redis-node
kubectl delete ClusterRoleBinding redis-operator, redis-node, redis-node-default
kubectl delete ServiceAccount redis-operator, redis-node
kubectl delete crd RedisCluster
