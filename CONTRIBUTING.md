# Contributing

By participating to this project, you agree to abide our [code of
conduct](/CODE_OF_CONDUCT.md).

## Setup your machine

`redis-operator` is written in [Go](https://golang.org/).

Prerequisites:

* `make`
* [Go 1.8+](https://golang.org/doc/install)
* [Docker](https://www.docker.com/)
* [Minikube](https://github.com/kubernetes/minikube)
* [Helm](https://helm.sh)

Clone `redis-operator` from source into `$GOPATH`:

```console
$ mkdir -p $GOPATH/src/github.com/amadeusitgroup
$ cd $_
$ git clone git@github.com:amadeusitgroup/redis-operator.git
Cloning into 'redis-operator'...
$ cd redis-operator
```

Install the build and lint dependencies:

```console
$ make setup
# install all developement dependencies
```

A good way of making sure everything is all right is running the test suite:

```console
$ make test
./go.test.sh
ok      github.com/amadeusitgroup/redis-operator/pkg/controller 1.520s  coverage: 0.0% of statements
ok      github.com/amadeusitgroup/redis-operator/pkg/controller/clustering      1.539s  coverage: 26.9% of statements
ok      github.com/amadeusitgroup/redis-operator/pkg/process    1.031s  coverage: 100.0% of statements
ok      github.com/amadeusitgroup/redis-operator/pkg/redis      1.570s  coverage: 26.7% of statements
ok      github.com/amadeusitgroup/redis-operator/pkg/redis/fake 1.026s  coverage: 84.7% of statements
```

## Test your change

You can create a branch for your changes and try to build from the source as you go:

```console
$ make build
CGO_ENABLED=0 go build -i -installsuffix cgo -ldflags '-w' -o bin/operator ./cmd/operator
CGO_ENABLED=0 go build -i -installsuffix cgo -ldflags '-w' -o bin/redisnode ./cmd/redisnode
```

When you are satisfied with the changes, we suggest you run:

```console
$ make test
$ make lint
gometalinter --vendor ./... -e pkg/client -e _generated -e test --deadline 2m -D gocyclo -D errcheck -D aligncheck
OK
```

Which runs all the linters and unit-tests.

### End-to-end test
you can run some end-to-end tests. For that you need to have a running Kubernetes cluster. You can use `minikube`. The deployment is done thanks to [Helm](https://helm.sh).

```console
$ minikube start --extra-config=apiserver.Authorization.Mode=RBAC
minikube start
Starting local Kubernetes v1.9.4 cluster...
Starting VM...
Getting VM IP address...
Moving files into cluster...
Setting up certs...
Connecting to cluster...
Setting up kubeconfig...
Starting cluster components...
Kubectl is now configured to use the cluster.
Loading cached images from config file.
$ kubectl create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default
clusterrolebinding "add-on-cluster-admin" created
$ helm init --wait
$HELM_HOME has been configured at /Users/clamoriniere/.helm.

Tiller (the Helm server-side component) has been installed into your Kubernetes Cluster.
Happy Helming!
```

Then you need to deploy the `redis-operator` on the kubernetes cluster.

```console
$ eval $(minikube docker-env)
# this will configure your docker cli, to target the minikube docker deamon
$ make TAG=latest container
CGO_ENABLED=0 GOOS=linux go build -i -installsuffix cgo -ldflags '-w' -o docker/operator/operator ./cmd/operator/main.go
CGO_ENABLED=0 GOOS=linux go build -i -installsuffix cgo -ldflags '-w' -o docker/redisnode/redisnode ./cmd/redisnode/main.go
$ make TAG=4.0 container
...
$ helm install --wait --name op chart/redis-operator
NAME:   op
LAST DEPLOYED: Tue Jan  9 23:41:13 2018
NAMESPACE: default
STATUS: DEPLOYED

RESOURCES:
==> v1beta1/Deployment
NAME               DESIRED  CURRENT  UP-TO-DATE  AVAILABLE  AGE
op-redis-operator  1        1        1           0          0s
```

When the `redis-operator` pod is up and running, you can start the e2e regression.

```console
$ cd test/e2e && go test --kubeconfig ~/.kube/config
Running Suite: RedisCluster Suite
=================================
Random Seed: 1515571968
Will run 1 of 1 specs

Ran 1 of 1 Specs in 35.222 seconds
SUCCESS! -- 1 Passed | 0 Failed | 0 Pending | 0 Skipped PASS
ok      github.com/amadeusitgroup/redis-operator/test/e2e       35.288s
```

## Create a commit

Commit messages should be well formatted.
Start your commit message with the type. Choose one of the following:
`feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `revert`, `add`, `remove`, `move`, `bump`, `update`, `release`

After a colon, you should give the message a title, starting with uppercase and ending without a dot.
Keep the width of the text at 72 chars.
The title must be followed with a newline, then a more detailed description.

Please reference any GitHub issues on the last line of the commit message (e.g. `See #123`, `Closes #123`, `Fixes #123`).

An example:

```
docs: Add example for --release-notes flag

I added an example to the docs of the `--release-notes` flag to make
the usage more clear.  The example is an realistic use case and might
help others to generate their own changelog.

See #284
```

## Submit a pull request

Push your branch to your `redis-operator` fork and open a pull request against the
master branch.
