#!/bin/bash
APP_DIR=$(cd `dirname $0`/../;pwd)
OS_TYPE=""
. $APP_DIR/scripts/common

if [[ "$OS_TYPE" =~ "Darwin" ]];then
    brew services stop etcd
else
    sudo systemctl stop etcd
fi

echo "stop etcd ok!" 

