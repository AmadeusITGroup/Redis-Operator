# Kubectl plugin

The Redis-operator kubectl plugin is here to help you visualise the status of your Redis-Cluster running in your cluster

if you don't know was is a kubectl plugin please the [official documentation](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/)

## installation

by default the plugin will be install in ```~/.kube/plugins```, if it is fine for you, you just need to run: ```make plugin```

if you want to install in another path you can run:

```shell
make plugin
```