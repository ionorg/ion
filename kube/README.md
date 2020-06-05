Kubernetes Config Readme
=================

***ALPHA QUALITY NOTE***
This Kubernetes configuration is basically un-tested. DO NOT USE IT IN PRODUCTION.

SFU Caveats
=======
+ Only 1 SFU is currently supported (pending ISLB Relay Feature)
+ The SFU currently opens 200 ports, 1 by 1, in sfu.yaml, so only 200 client streams are supported
+ We tried to use `hostNetwork: true` and `dnsPolicy: ClusterFirstThenHost` instead but we haven't yet seen a successful ICE connection this way


Features
========
If you install nats, redis and etcd from the helm charts with metrics.enabled, you get
prometheus service health monitoring for free!


Development Notes
=================
+ It should be upgraded to a Helm chart ASAP; I have never done this, I am learning
+ Tested locally on k3s, but you *must* have local SSL certs working
