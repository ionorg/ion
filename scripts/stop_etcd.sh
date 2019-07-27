#!/bin/bash

APP_DIR=$(cd `dirname $0`/../; pwd)
ETCD_DIR=$APP_DIR/bin/etcd
EXE=etcd
cd $APP_DIR

PID_FILE=$ETCD_DIR/$EXE.pid  #pid file

help()
{
    echo ""
    echo "stop script version: 0.1"
    echo "Usage: sh stop.sh [-h]"
    echo ""
}

while getopts "p:h" arg
do
    case $arg in
        h)
            help;
            exit 0
            ;;
        ?)
            echo "No argument needed. Will ignore them all!"
            ;;
    esac
done

echo "stop process..."
PID=`cat $PID_FILE`
if [ ! -n "$PID" ]; then
    echo "pid not exist"
    exit 1;
fi
SUB_PIDS=`pgrep -P $PID`
if [ -n "$SUB_PIDS" ]; then
    GRANDSON_PIDS=`pgrep -P $SUB_PIDS`
fi

echo "kill $PID $SUB_PIDS $GRANDSON_PIDS"
kill -9 $PID $SUB_PIDS $GRANDSON_PIDS
if [[ $? -eq 0 ]];then
    rm -f $PID_FILE
fi
echo "finish stop process..."

