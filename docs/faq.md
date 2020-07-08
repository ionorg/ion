## This a question and answer page!
### 1.Does ion create room?
No, You can DIY a room service
### 2.Can ion block users to join a room?
No, You can DIY a websocket reverse proxy for biz which support auth(jwt etc..)
### 3.Is this just a SFU media server which we can self host?
No, all ion service can be deloyed online. You must open the port which they need.
### 4.Does it include its own TURN which is used by our host server?
No. 
### 5.Some node panic when startup (islb etc..)
`2020-04-28 21:15:52.809 ERR read tcp 127.0.0.1:65507->127.0.0.1:4440: read: connection reset by peer`

Make sure your vpn is closed
### 6.How to deploy on cloud by scripts?
1) git clone https://github.com/pion/ion
2) cd ion, do some change:

set the port range:
```
portrange = [50000, 60000]
```
uncomment the two lines:
```
[[webrtc.iceserver]]
urls = ["stun:stun.stunprotocol.org:3478"]
```
add two lines to:
sdk/js/demo/webpack.config.js
```
--- a/sdk/js/demo/webpack.config.js
+++ b/sdk/js/demo/webpack.config.js
@@ -32,5 +32,7 @@ module.exports = {
     contentBase: './dist',
     hot: true,
     host: '0.0.0.0',
+     disableHostCheck: true,
+     port: 443,
   }
 };
```
3) make sure these ports is opened on your cloud server firewall!!
```
443
8443
50000-60000
```
4) run all modules

first, put your cert.pem and key.pem in configs

second, run
```
./scripts/allStart.sh
```
then, chat with https://yourdomain

### 7.How to deploy on cloud by docker
The steps I took to get the docker version running on a fresh AWS vm
Names will be different for GCP and Azure, but similar concept
1) Create vm
2) Export ports 80, 443 and 5000-5200 as detiled in the readme
3) Map elastic/external ip to the vm
4) Configure dns records to point to new ip. THIS IS NEEDED LATER
5) Clone Ion and checkout docker branch
6) Modify docker-compose.yml following the readme instructions and with the domain you mapped earlier.
```
export WWW_URL=yourdomain
export ADMIN_EMAIL=youremail
docker-compose pull
docker-compose up
```
7) docker-compose up
8) chat with: https://yourdomain:8080