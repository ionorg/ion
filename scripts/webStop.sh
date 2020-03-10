#!/bin/bash

APP_DIR=$(cd `dirname $0`/../; pwd)

PID=`cat $APP_DIR/configs/node.pid`
if [ ! -n "$PID" ]; then
    echo "pid not exist"
    exit 1;
fi
SUB_PIDS=`pgrep -P $PID`
if [ -n "$SUB_PIDS" ]; then
    GRANDSON_PIDS=`pgrep -P $SUB_PIDS`
fi

echo "kill $PID $SUB_PIDS $GRANDSON_PIDS"
kill $PID $SUB_PIDS $GRANDSON_PIDS
echo "stop web ok"

