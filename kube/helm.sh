#!/usr/bin/env bash
set -e

helm install ion-nats bitnami/nats --namespace=ion \
  --set auth.enabled=false \
  --set clusterDomain=netp.tech \
  --set metrics.enabled=true | tee -a helm.log

helm install ion-redis bitnami/redis --namespace=ion \
  --set usePassword=false \
  --set clusterDomain=netp.tech \
  --set metrics.enabled=true | tee -a helm.log

helm install ion-etcd bitnami/etcd --namespace=ion \
  --set auth.rbac.enabled=false \
  --set clusterDomain=netp.tech \
  --set metrics.enabled=true | tee -a helm.log
