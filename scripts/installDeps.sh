#!/bin/bash
APP_DIR=$(cd `dirname $0`/../;pwd)
OS_TYPE=""
. $APP_DIR/scripts/common


# Ubuntu
if [[ "$OS_TYPE" =~ "Ubuntu" ]];then
    sudo apt-get install -y etcd redis-server nodejs-legacy npm
    wgetdl https://github.com/nats-io/nats-server/releases/download/v2.1.4/nats-server-v2.1.4-linux-amd64.zip
    uz nats-server-v2.1.4-linux-amd64.zip
    sudo cp nats-server-v2.1.4-linux-amd64/nats-server /usr/bin
    sudo npm install -g n
    sudo n stable 
fi

# Centos7
if [[ "$OS_TYPE" =~ "CentOS" ]];then
    npm config set registry http://registry.cnpmjs.org/
    sudo yum install epel-release
    sudo yum install -y etcd redis nodejs
    wgetdl https://github.com/nats-io/nats-server/releases/download/v2.1.4/nats-server-v2.1.4-amd64.rpm
    sudo rpm -ivh nats-server-v2.1.4-amd64.rpm
fi

# Mac
if [[ "$OS_TYPE" =~ "Darwin" ]];then
    brew install etcd redis nodejs nats-server
fi

