#!/bin/bash
set -eux

APP_DIR=$(cd `dirname $0`/../;pwd)
OS_TYPE=""
. $APP_DIR/scripts/common


if [[ "$OS_TYPE" =~ "Darwin" ]];then
    brew services stop redis
else
    sudo systemctl stop redis
fi

echo "stop redis ok!"

