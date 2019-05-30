# SFU

## Architecture

![arch](arch.png)

## Contributing
* [adwpc](https://github.com/adwpc) - *pion sfu server*
* [cloudwebrtc](https://github.com/cloudwebrtc) - *pion sfu sdk*

## Roadmap
[Projects](https://github.com/pion/ion/projects/1)

## Project status
[![Stargazers over time](https://starchart.cc/pion/ion.svg)](https://starchart.cc/pion/ion)

## How to use
### 1. make key.pem|cert.pem
```
./scripts/makekey.sh
```
### 2. build web app
```
cd sdk/js
npm i
```
### 3. build ion
```
./scripts/build.sh
```
### 4. start web app
```
cd sdk/js
npm start
```
### 5. start ion
```
./scripts/start.sh
```
### 6. let's chat
Open this url with chrome

```
https://yourip:8080
```
