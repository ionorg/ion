#!/bin/bash
APP_DIR=$(cd `dirname $0`/../../; pwd)

sudo apt-get install  -y etcd redis-server rabbitmq-server nodejs-legacy npm

sudo npm install -g n
sudo n stable 

npm config set registry http://registry.cnpmjs.org/

cd $APP_DIR/sdk/js
npm i
