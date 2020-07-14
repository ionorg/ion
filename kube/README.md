# Kubernetes Config Readme

***ALPHA QUALITY NOTE***
This Kubernetes configuration is basically un-tested. DO NOT USE IT IN PRODUCTION.

## Requirements
1. Helm3 - We use helm3 to install all components of ion. 
2. A public IP and valid subdomain to use for `nginx-ingress`; it must be a valid resolving subdomain with a publicly accessible IP for the TLS to provision. 
3. `nginx-ingress` with TLS is *required*; there are many possible ways to set this up, depending on your Kubernetes provider.  For lets-encrypt, install [cert-manager](https://cert-manager.io/docs/) inside the cluster. If you use a different ingress or don't support automatic TLS, you must ensure the Web service has TLS configured; the video chat will always fail to connect over HTTP.

Getting a `LoadBalancer` with a public IP and setting up TLS are well-documented roadblocks for many new Kubernetes users. Rather than testing on a local machine on a home network behind a router, it might be easier to provision a kubernetes cluster with proper `LoadBalancer` support.

## Usage

This helm chart will install all dependencies (redis, etcd, and nats) as well as all the ion components (sfu, biz, islb, web).  It creates an ingress configuration for the domain you configure at install time.

```
export RELEASE_NAME=ion # Helm Release Name
helm install $RELEASE_NAME ion \
    --namespace pion \
    --set ingress.domain=sfu.example.com  # This domain should resolve to your nginx-controller's public IP address
```


### GKE
The SFU Deployment requires host networking to expose the range orf UDP ports used for RTP traffic (due to the lack of support for UDP ranges in kubernetes services).  This requires a custom firewall rule that will expose traffic to the kubernetes nodes for the UDP range set in `ion/templates/config.yaml`.  This can be created for your default-network in GCP with the following gcloud command.

```
gcloud compute firewall-rules create ion-webrtc --allow udp:5000-52000,udp:6666
```


## SFU Caveats
+ Only 1 SFU is currently supported (pending ISLB Relay Feature)
+ SFU is currently configured as a Deployment(scale=1); this will be changed to a DaemonSet (1 pod per node) once Relay is supported

