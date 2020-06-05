#!/usr/bin/env bash

kubectl apply -f config.yaml
kubectl apply -f biz.yaml
kubectl apply -f islb.yaml
kubectl apply -f sfu.yaml
kubectl apply -f web.yaml

kubectl apply -f ingress.yaml
