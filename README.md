# SFU

## Architecture

![arch](arch.png)

## Contributing
* [adwpc](https://github.com/adwpc) - *Original Author - ion sfu server*
* [cloudwebrtc](https://github.com/cloudwebrtc) - *Original Author - ion sfu sdk*

## Roadmap

[Projects](https://github.com/pion/ion/projects/1)

## Project status
[![Stargazers over time](https://starchart.cc/pion/ion.svg)](https://starchart.cc/pion/ion)

## How to use
### 1. make key.pem|cert.pem
```
./scripts/makekey.sh
```
### 2. install deps
```
./scripts/installDeps.sh
```
### 3. build web app
```
cd sdk/js
npm i
```
### 4. build ion
```
./scripts/build.sh
```
### 5. start web app
```
cd sdk/js
npm start
```
### 6. start etcd
```
./scripts/start_etcd.sh
```
### 7. start ion
```
./scripts/start.sh
```
### 8. let's chat
Open this url with chrome

```
https://yourip:8080
```


