#!/usr/bin/env bash
set -e

### Data Services

helm repo add bitnami https://charts.bitnami.com/bitnami

helm install ion-nats bitnami/nats --namespace=ion \
  --set auth.enabled=false \
  --set metrics.enabled=true | tee -a helm.log

helm install ion-redis bitnami/redis --namespace=ion \
  --set usePassword=false \
  --set metrics.enabled=true | tee -a helm.log

helm install ion-etcd bitnami/etcd --namespace=ion \
  --set auth.rbac.enabled=false \
  --set metrics.enabled=true | tee -a helm.log


### Cert Manager

kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v0.15.1/cert-manager.yaml

kubectl apply -f cert-issuer.yaml
