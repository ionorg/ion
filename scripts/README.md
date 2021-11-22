
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

It will start all services we need, checking

```
./scripts/all status
```

Next Time, just restart:

```
./scripts/all restart
```

## 4. How to run a ion module

Usage: ./service {start|stop} {app-room|signal|islb|sfu}

example:
```
./scripts/service start app-room
```

## 5. How to run a deps module

Usage: ./deps {start|stop} {nats-server|redis}

example:
```
./scripts/deps start redis
```
