#!/bin/bash

APP_DIR=$(cd `dirname $0`/../; pwd)
cd $APP_DIR/sdk/js

nohup npm start 2>&1& echo $! > $APP_DIR/logs/node.pid
echo "start web ok"

