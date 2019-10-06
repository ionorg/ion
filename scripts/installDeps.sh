#!/bin/bash
APP_DIR=$(cd `dirname $0`/../; pwd)

sudo yum install epel-release
sudo yum install -y etcd redis rabbitmq-server nodejs
