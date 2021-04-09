## docker-compose Quick Start
#### 1. Run

```
git clone https://github.com/pion/ion

docker network create ionnet

cd ion

docker-compose -f docker-compose.stable.yml up -d
```

#### 3. Expose Ports

Ensure the ports `5000-5200/udp` are exposed or forwarded for the SFU service. If you are on a cloud provider like GCP or AWS, ensure your SFU instances have a publicly routable IP, and open those ports on the relevant firewalls or security groups.


#### 4. UI (optional)

Head over to [Ion Web App](https://github.com/pion/ion-app-web) to bring up the front end.

The web app repo also contains examples of exposing the ion biz websocket via reverse proxy with automatic SSL.

For dev and more options see the wiki

* [Development](https://github.com/pion/ion/tree/master/docs)
