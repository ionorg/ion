# SFU

## Architecture

![arch](arch.png)

## Contributing
* [adwpc](https://github.com/adwpc) - *pion sfu server*
* [cloudwebrtc](https://github.com/cloudwebrtc) - *pion sfu sdk*

## Roadmap
[Projects](https://github.com/pion/sfu/projects/1)

## Project status
[![Stargazers over time](https://starchart.cc/pion/sfu.svg)](https://starchart.cc/pion/sfu)

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
### 3. build sfu
```
./scripts/build.sh
```
### 4. start web app
```
cd sdk/js
npm start
```
### 5. start sfu
```
./scripts/start.sh
```
### 6. let's chat
Open this url with chrome

```
https://yourip:8080
```
