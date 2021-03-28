
## scripts

* Support os: Ubuntu MacOS CentOS

## 1. Install deps

```
./scripts/deps_inst
```

It will install all depend modules, support mac, ubuntu, centos
Check these modules installed:nats-server redis

## 2. Make key

```
./scripts/key
```

It will generate key files to configs

## 3. Run all services

First Time

```
./scripts/all start
```

It will start all services we need

Next Time, just restart:

```
./scripts/all restart
```

## 4. How to run a ion module

Usage: ./service {start|stop} {biz|islb|sfu|avp}

example:
```
./scripts/service start biz
```

## 5. How to run a deps module

Usage: ./deps {start|stop} {nats-server|redis}

example:
```
./scripts/deps start redis
```
