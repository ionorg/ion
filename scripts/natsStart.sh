#!/bin/bash

APP_DIR=$(cd `dirname $0`/../; pwd)
cd $APP_DIR
OS_TYPE=""
. $APP_DIR/scripts/common

mkdir -p $APP_DIR/logs
EXE=nats-server
COMMAND=$EXE
#CONFIG=$APP_DIR/configs/
PID_FILE=$APP_DIR/configs/nats-server.pid
LOG_FILE=$APP_DIR/logs/nats-server.log

help()
{
    echo ""
    echo "start script"
    echo "Usage: ./natsStart.sh [-h]"
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
            echo "No argument needed. Ignore them all!"
            ;;
    esac
done


count=`ps -ef |grep " $COMMAND " |grep -v "grep" |wc -l`
if [ 0 != $count ];then
    ps aux | grep " $COMMAND " | grep -v "grep"
    echo "already start"
    exit 1;
fi

# if [ ! -r $CONFIG ]; then
    # echo "$CONFIG not exsist"
    # exit 1;
# fi
if [[ "$OS_TYPE" =~ "Darwin" ]];then
    brew services start nats-server
else
    ## run command
    echo "nohup $COMMAND >>$LOG_FILE 2>&1 &"
    nohup $COMMAND >>$LOG_FILE 2>&1 &
    pid=$!
    echo "$pid"
    echo "$pid" > $PID_FILE
    rpid=`ps aux | grep $pid |grep -v "grep" | awk '{print $2}'`
    if [[ $pid != $rpid ]];then
        echo "start failly. $pid $rpid"
        # rm  $PID_FILE
        exit 1
    fi
fi
