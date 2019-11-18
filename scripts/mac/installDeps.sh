#!/bin/bash
APP_DIR=$(cd `dirname $0`/../../; pwd)

brew install etcd redis rabbitmq nodejs
cd $APP_DIR/sdk/js
npm i
