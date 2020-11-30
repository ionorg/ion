#!/bin/bash
set -eux

APP_DIR=$(cd `dirname $0`/../;pwd)
OS_TYPE=""
. $APP_DIR/scripts/common


if [[ "$OS_TYPE" =~ "Darwin" ]];then
    brew services start redis
else
    sudo systemctl start redis
fi

echo "start redis ok!"
