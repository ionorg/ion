Kubernetes Config Readme
=================

***ALPHA QUALITY NOTE***
This Kubernetes configuration is basically un-tested. DO NOT USE IT IN PRODUCTION.

Features
========
If you install nats, redis and etcd from the helm charts with metrics.enabled, you get
prometheus service health monitoring for free!

Requirements
============
1. Helm3 - We use helm3 to install NATS/etcd/redis in your namespace
2. `nginx-ingress` with automatic TLS is *required*. Instructions for installing this depend on your Kubernetes provider and are outside the scope of this document.

Usage
=====
0. (recommended) Read every file in this directory before deploying. They are short and crucial. Understand them.
1. Install NATS/etcd/redis by running `bash helm.sh`
2. Update ingress.yaml to add your domain for HTTPS; it must be a valid resolving subdomain. TLS is required for the ion frontend, video will not connect without it.
3. Install the ion stack (SFU/ISLB/BIZ/AVP nodes and the WEB service) by running `bash apply.sh`

SFU Caveats
=======
+ Only 1 SFU is currently supported (pending ISLB Relay Feature)
+ SFU is currently configured as a Deployment; this will be changed to a DaemonSet once Relay is supported

Development Notes
=================
+ It should be upgraded to a Helm chart ASAP; I have never done this, I am learning
+ Tested locally on k3s, but you *must* have local SSL certs working (which can be hard to setup)
