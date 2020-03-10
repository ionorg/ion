#!/bin/bash
APP_DIR=$(cd `dirname $0`/../;pwd)
OS_TYPE=""
. $APP_DIR/scripts/common


if [[ "$OS_TYPE" =~ "Darwin" ]];then
    brew services start etcd
else
    sudo systemctl start etcd
fi

echo "start etcd ok!"
