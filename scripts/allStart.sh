#!/bin/bash
set -eux

APP_DIR=$(cd `dirname $0`/../; pwd)
cd $APP_DIR
mkdir -p $APP_DIR/logs

help()
{
    echo ""
    echo "start script"
    echo "Usage: ./allRestart.sh [-h]"
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

# run command
# echo "------------etcd--------------"
# $APP_DIR/scripts/etcdStart.sh

# echo "------------nats-server--------------"
# $APP_DIR/scripts/natsStart.sh

# echo "------------redis--------------"
# $APP_DIR/scripts/redisStart.sh

docker-compose up -d nats redis etcd

echo "------------islb--------------"
$APP_DIR/scripts/islbStart.sh

echo "------------biz--------------"
$APP_DIR/scripts/bizStart.sh

echo "------------sfu--------------"
$APP_DIR/scripts/sfuStart.sh

echo "--------------------------"



