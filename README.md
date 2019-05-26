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
### 1. install signal server and make key.pem|cert.pem
```
cd signal-server
./install.sh
```
### 2. install web app
```
cd pion-sfu-sdk
npm i
```
### 3. start signal server
```
cd signal-server
./start.sh
```
### 4. start web app
```
cd pion-sfu-sdk
npm start
```
### 5. start sfu
```
go build
./sfu
```
### 6. let's chat
Open this url with chrome

```
https://yourip:3666
```
