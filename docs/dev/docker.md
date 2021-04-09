## How to use

Docker commands require the ionnet docker network

First run:

```
docker network create ionnet
```

### Deployment

#### 1. Clone

```
git clone https://github.com/pion/ion
```

#### 2. Setup

Firstly pull images. Skip this command if you want build images locally

```
docker-compose pull
```

#### 3. Run

```
docker-compose up
```

#### 4. Expose Ports

(Skip if only exposing locally)

Ensure the following ports are exposed or forwarded.

```
5000-5200/udp
```

#### 5. Chat

Head over to [Ion Web App](https://github.com/pion/ion-app-web) to bring up the front end.

The web app repo also contains examples of exposing the ion biz websocket via reverse proxy with automatic SSL.