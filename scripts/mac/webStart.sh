#!/bin/bash

APP_DIR=$(cd `dirname $0`/../../; pwd)
cd $APP_DIR/sdk/js
npm i

nohup npm start 2>&1& echo $! > $APP_DIR/configs/node.pid
echo "start web ok"

