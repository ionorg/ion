#!/usr/bin/env bash
PID_FILE="centrifugo.pid"


echo "stopping centrifugo"
PID=`cat $PID_FILE`
if [ ! -n "$PID" ]; then
    echo "pid not exist"
    exit 1;
fi

echo "kill -9 $PID"
kill -9 $PID
rm -rf $PID_FILE
echo "stop ok"
