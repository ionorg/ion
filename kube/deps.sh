#!/usr/bin/env bash
set -e

test -z "$1" && NS="ion" || NS="$1"

echo "Installing data services into namespace $NS..."

kubectl create namespace $NS || true

### Data Services

helm repo add bitnami https://charts.bitnami.com/bitnami

helm install ion-nats bitnami/nats --namespace=$NS \
  --set auth.enabled=false \
  --set metrics.enabled=true | tee -a helm.log

helm install ion-redis bitnami/redis --namespace=$NS \
  --set usePassword=false \
  --set metrics.enabled=true | tee -a helm.log

helm install ion-etcd bitnami/etcd --namespace=$NS \
  --set auth.rbac.enabled=false \
  --set metrics.enabled=true | tee -a helm.log


### Cert Manager

kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v0.15.1/cert-manager.yaml

kubectl apply -f cert-issuer.yaml
