Kubernetes Config Readme
=================

***ALPHA QUALITY NOTE***
This Kubernetes configuration is basically un-tested. DO NOT USE IT IN PRODUCTION.

Requirements
============
1. Helm3 - We use helm3 to install NATS/etcd/redis in your namespace
2. A public IP and valid subdomain to use for `nginx-ingress`; it must be a valid resolving subdomain with a publicly accessible IP for the TLS to provision. 
3. `nginx-ingress` with TLS is *required*; there are many possible ways to set this up, depending on your Kubernetes provider. If you use a different ingress or don't support automatic TLS, you must ensure the Web service has TLS configured; the video chat will always fail to connect over HTTP.

Getting a `LoadBalancer` with a public IP and setting up TLS are well-documented roadblocks for many new Kubernetes users. Rather than testing on a local machine on a home network behind a router, it might be easier to provision a kubernetes cluster with proper `LoadBalancer` support.

Usage
=====
0. (recommended) Read every file in this directory before deploying. They are short and crucial. Understand them.
1. `kubectl create namespace ion` -- You can use another namespace but you'll need to update parts of the `grafana_charts` in step 5
2. Update `ingress.yaml` and `cert-manager.yaml` to add your domain and email address; you might want to update the `ingress.class` if you are using `traefik` for ingress (like `k3s` does by default).
3. Install NATS/etcd/redis and cert-manager by running `bash deps.sh`; you can comment out `cert-manager` if it's already installed.
4. Install the ion stack (SFU/ISLB/BIZ/AVP nodes and the WEB service) by running `bash apply.sh`
5. [optional] Add the `grafana` charts from `docs/grafana_charts/`; you can install a portable grafana in the current namespace just by running `helm install grafana bitnami/grafana`

SFU Caveats
=======
+ Only 1 SFU is currently supported (pending ISLB Relay Feature)
+ SFU is currently configured as a Deployment(scale=1); this will be changed to a DaemonSet (1 pod per node) once Relay is supported

Development Notes
=================
+ It should be upgraded to a Helm chart ASAP; I have never done this, I am learning
+ Tested locally on k3s, but you *must* have local SSL certs working