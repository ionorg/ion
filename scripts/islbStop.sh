#!/bin/bash
set -eux

APP_DIR=$(cd `dirname $0`/../; pwd)
cd $APP_DIR

PID_FILE=$APP_DIR/configs/islb.pid  #pid file, default: worker.pid

help()
{
    echo ""
    echo "stop script"
    echo "Usage:./islbStop.sh [-h]"
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
            echo "No argument needed. Ignore them all!"
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
kill $PID $SUB_PIDS $GRANDSON_PIDS
rm -rf $PID_FILE
echo "finish stop process..."

