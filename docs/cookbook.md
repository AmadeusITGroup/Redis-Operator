# Minikube

## Prerequisit

- minikube version >= 0.20
- kubectl >= 1.7
- helm
- golang >= 1.8
- docker >= 1.10
- make
- git

## Commands

Project and environment

```console
$ REPO="https://github.com/amadeusitgroup/redis-operator.git"
$ FOLDER="$HOME/tmp/redis-operator"

$ mkdir -p $FOLDER/src/github.com/amadeusitgroup/redis-operator
$ cd $FOLDER
$ export GOPATH=`pwd`
$ git clone $REPO $GOPATH/src/github.com/amadeusitgroup/redis-operator
$ cd $GOPATH/src/github.com/amadeusitgroup/redis-operator
```

install the kubectl rediscluster plugin (more info [here](./kubectl-plugin.md))

```console
$ make plugin
```

start minikube with RBAC activated

```console
$ minikube start --extra-config=apiserver.Authorization.Mode=RBAC
```

Create the missing rolebinding for k8s dashboard

```console
$ kubectl create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default
clusterrolebinding "add-on-cluster-admin" created
```

Create the cluster role binding for the helm tiller

```console
$ kubectl create clusterrolebinding tiller-cluster-admin  --clusterrole=cluster-admin --serviceaccount=kube-system:default
clusterrolebinding "tiller-cluster-admin" created
```

Init the helm tiller

```console
$ helm init --wait
HELM_HOME has been configured at /Users/<user>/.helm.
Tiller (the Helm server-side component) has been installed into your Kubernetes Cluster.
Happy Helming!
```

check if Helm is running properly

```console
checkhelm(){ CHECKHELM=$(kubectl get pod -n kube-system -l app=helm,name=tiller | grep "1/1" | wc -l); }
checkhelm; while [ ! $CHECKHELM -eq 1 ]; do sleep 1; checkhelm; done
```

Build docker images

```console
$ eval $(minikube docker-env)
$ make TAG=latest container
CGO_ENABLED=0 GOOS=linux go build -i -installsuffix cgo -ldflags '-w' -o docker/operator/operator ./cmd/operator/main.go
...
```

Install the redis-operator thanks to helm

```console
$ helm install --wait -n operator chart/redis-operator
NAME:   operator
LAST DEPLOYED: Thu Feb 15 14:53:39 2018
NAMESPACE: default
STATUS: DEPLOYED

RESOURCES:
==> v1beta1/Deployment
NAME                     DESIRED  CURRENT  UP-TO-DATE  AVAILABLE  AGE
operator-redis-operator  1        0        0           0          0s

==> v1beta1/ClusterRole
NAME            AGE
redis-operator  0s

==> v1beta1/ClusterRoleBinding
NAME            AGE
redis-operator  0s

==> v1/ServiceAccount
NAME            SECRETS  AGE
redis-operator  1        0s
```

Monitor that the operator is running properly.

```console
$ kubectl get pods -w
NAME                                       READY     STATUS    RESTARTS   AGE
operator-redis-operator-5d64589b66-9rwsx   1/1       Running   0          1m
```

# Create a redis cluster

```console
$ helm install --wait -n mycluster chart/redis-cluster --set numberOfMaster=3 --set replicationFactor=1
NAME:   mycluster
LAST DEPLOYED: Thu Feb 15 14:56:35 2018
NAMESPACE: default
STATUS: DEPLOYED

RESOURCES:
==> v1beta1/ClusterRoleBinding
NAME        AGE
redis-node  0s

==> v1/ServiceAccount
NAME        SECRETS  AGE
redis-node  1        0s

==> v1alpha1/RedisCluster
NAME       AGE
mycluster  0s

==> v1beta1/ClusterRole
redis-node  0s
```

Monitor the redis-cluster creation

```console
$ watch kubectl plugin rediscluster --rc myCluster
Every 2.0s: kubectl plugin rediscluster
 NAME       NAMESPACE  PODS   STATUS  NB MASTER  REPLICATION
 mycluster  default    6/6/6  OK      3/3        1-1/1
```

check the cluster status thanks to redis command `cluster info`:

```console
$ kubectl exec $(kubectl get pod -l redis-operator.io/cluster-name=mycluster -o jsonpath="{.items[0].metadata.name}") -- redis-cli cluster info
cluster_state:ok
cluster_slots_assigned:16384
cluster_slots_ok:16384
cluster_slots_pfail:0
cluster_slots_fail:0
cluster_known_nodes:6
cluster_size:3
cluster_current_epoch:4
cluster_my_epoch:1
cluster_stats_messages_ping_sent:7003
cluster_stats_messages_pong_sent:6990
cluster_stats_messages_sent:13993
cluster_stats_messages_ping_received:6985
cluster_stats_messages_pong_received:7003
cluster_stats_messages_meet_received:5
cluster_stats_messages_received:13993

$ kubectl exec $(kubectl get pod -l redis-operator.io/cluster-name=mycluster -o jsonpath="{.items[0].metadata.name}") -- redis-cli cluster nodes
7e4d1efc89a8230fe67c039bdb4101d44585a56a 172.17.0.9:6379@16379 slave f73d9e04184a3ab5a83f7073b28f02d481818f6c 0 1518704528057 0 connected
147a54a784e46152ab4de0b6351e9d7c0908ad49 172.17.0.7:6379@16379 master - 0 1518704527250 1 connected 5462-10923
f73d9e04184a3ab5a83f7073b28f02d481818f6c 172.17.0.8:6379@16379 master - 0 1518704527250 0 connected 0-5461
42e36bbbcfb49eedb47220d345c578935c771094 172.17.0.11:6379@16379 myself,slave 147a54a784e46152ab4de0b6351e9d7c0908ad49 0 1518704527000 0 connected
5dfadc738994f9b4ae8ae4af90dba0387ec676b2 172.17.0.10:6379@16379 slave caa70778d7b52f43a1145e6f19ccd31194b70c32 0 1518704527149 2 connected
caa70778d7b52f43a1145e6f19ccd31194b70c32 172.17.0.6:6379@16379 master - 0 1518704527755 2 connected 10924-16383
```

## cleanup your environement

delete the redis cluster

```console
$ helm delete mycluster
release "mycluster" deleted
```

delete the redis-operator

```console
$ helm delete operator
release "operator" deleted
```

all pods should have been deleted
```console
$ kubectl get pods
No resources found.
```
