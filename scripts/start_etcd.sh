#!/bin/bash

APP_DIR=$(cd `dirname $0`/../; pwd)
cd $APP_DIR
ETCD_DIR=$APP_DIR/bin/etcd
mkdir -p $ETCD_DIR/logs
EXE=etcd
COMMAND=$ETCD_DIR/$EXE
STOP=$APP_DIR/scripts/stop_etcd.sh
PID_FILE=$ETCD_DIR/$EXE.pid             #pid file
LOG_FILE=$ETCD_DIR/logs/$EXE.log         #console log

help()
{
    echo ""
    echo "start script version: 0.1"
    echo "Usage: sh start.sh [-h]"
    echo ""
}

while getopts "h" arg
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


count=`ps -ef |grep " $COMMAND " |grep -v "grep" |wc -l`
if [ 0 != $count ];then
    ps aux | grep " $COMMAND " | grep -v "grep"
    echo "$EXE already start"
    exit 1;
fi

## run command
cd $ETCD_DIR
eval $ETCD_DIR/etcd --auth-token "jwt,pub-key=$ETCD_DIR/app.rsa.pub,priv-key=$ETCD_DIR/app.rsa,sign-method=HS256" > $LOG_FILE 2>&1 &

pid=$!
echo "$pid" > "$PID_FILE"
sleep 1
rpid=`ps aux | grep $pid |grep -v "grep" | awk '{print $2}'`
if [[ $pid != $rpid ]];then
	echo "start failed!"
    rm  $PID_FILE
	exit 1
fi
echo "start ok!"





