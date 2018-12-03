#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o pipefail
set -x

# Download kubectl, which is a requirement for using minikube.
curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.9.4/bin/linux/amd64/kubectl && chmod +x kubectl && sudo mv kubectl /usr/local/bin/
# Download minikube.
curl -Lo minikube https://storage.googleapis.com/minikube/releases/v0.25.2/minikube-linux-amd64 && chmod +x minikube && sudo mv minikube /usr/local/bin/
sudo minikube start --vm-driver=none --kubernetes-version=v1.9.4 --extra-config=apiserver.Authorization.Mode=RBAC
# Fix the kubectl context, as it's often stale.
minikube update-context
sudo ln -s /usr/local/bin/kubectl /usr/local/bin/ctl