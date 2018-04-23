# End to end Tests

you need to have a Kubernetes cluster up and running with service-catalog installed.

if you need to install service catalog you can do:

```shell
helm repo add svc-cat https://svc-catalog-charts.storage.googleapis.com
helm search service-catalog
helm install --wait svc-cat/catalog  --version 0.1.9 --name catalog --namespace catalog
```

then:

```shell
$go test -c
```

which will compile the `e2e.test` executable in your current directory. Then

```shell
./e2e.test --kubeconfig=$HOME/.kube/config

```

will start the e2e test....
