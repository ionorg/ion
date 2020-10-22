#!/bin/bash
set -eux

APP_DIR=$(cd `dirname $0`/../; pwd)
cd $APP_DIR
mkdir -p $APP_DIR/logs
EXE=sfu
COMMAND=$APP_DIR/bin/$EXE
CONFIG=$APP_DIR/configs/$EXE.toml
PID_FILE=$APP_DIR/configs/$EXE.pid
LOG_FILE=$APP_DIR/logs/$EXE.log

help()
{
    echo ""
    echo "start script"
    echo "Usage: ./${EXE}Start.sh [-h]"
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

if [ ! -r $CONFIG ]; then
    echo "$CONFIG not exsist"
    exit 1;
fi

## build first
cd $APP_DIR/cmd/$EXE
echo "go build -o $COMMAND"
go build -o $COMMAND
cd $APP_DIR

## run command
echo "nohup $COMMAND -c $CONFIG >>$LOG_FILE 2>&1 &"
nohup $COMMAND -c $CONFIG >>$LOG_FILE 2>&1 &
pid=$!
echo "$pid" > $PID_FILE
rpid=`ps aux | grep $pid |grep -v "grep" | awk '{print $2}'`
if [[ $pid != $rpid ]];then
	echo "start failly."
    rm  $PID_FILE
	exit 1
fi

