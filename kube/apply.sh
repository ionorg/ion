#!/usr/bin/env bash

set -e

test -z "$1" && NS="ion" || NS="$1"

echo "Installing ion services into namespace $NS..."

kubectl -n $NS apply -f config.yaml
kubectl -n $NS apply -f biz.yaml
kubectl -n $NS apply -f islb.yaml
kubectl -n $NS apply -f sfu.yaml
kubectl -n $NS apply -f web.yaml

kubectl -n $NS apply -f ingress.yaml
