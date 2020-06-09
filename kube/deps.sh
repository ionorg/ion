#!/usr/bin/env bash
set -e

test -z "$1" && NS="ion" || NS="$1"

echo "Installing data services into namespace $NS..."

### Data Services

helm repo add bitnami https://charts.bitnami.com/bitnami

helm install -n $NS ion-nats bitnami/nats --namespace=ion \
  --set auth.enabled=false \
  --set metrics.enabled=true | tee -a helm.log

helm install -n $NS ion-redis bitnami/redis --namespace=ion \
  --set usePassword=false \
  --set metrics.enabled=true | tee -a helm.log

helm install -n $NS ion-etcd bitnami/etcd --namespace=ion \
  --set auth.rbac.enabled=false \
  --set metrics.enabled=true | tee -a helm.log


### Cert Manager

kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v0.15.1/cert-manager.yaml

kubectl apply -f cert-issuer.yaml
