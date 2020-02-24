#!/bin/bash
APP_DIR=$(cd `dirname $0`/../;pwd)
OS_TYPE=""
. $APP_DIR/scripts/common


# centos7
if [[ "$OS_TYPE" =~ "Ubuntu" ]];then
    sudo apt-get install -y etcd redis-server rabbitmq-server nodejs-legacy npm
    #TODO add nats install
    sudo npm install -g n
    sudo n stable 
fi

if [[ "$OS_TYPE" =~ "CentOS" ]];then
    npm config set registry http://registry.cnpmjs.org/
    sudo yum install epel-release
    sudo yum install -y etcd redis rabbitmq-server nodejs
    wgetdl https://github.com/nats-io/nats-server/releases/download/v2.1.4/nats-server-v2.1.4-amd64.rpm
    sudo rpm -ivh nats-server-v2.1.4-amd64.rpm
fi

if [[ "$OS_TYPE" =~ "Darwin" ]];then
    brew install etcd redis rabbitmq nodejs nats-server
fi

exit

cd $APP_DIR/sdk/js
npm i
