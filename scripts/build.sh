#!/bin/bash

APP_DIR=$(cd `dirname $0`/../; pwd)
cd $APP_DIR
EXE=sfu
COMMAND=$APP_DIR/bin/$EXE

help()
{
    echo ""
    echo "build script"
    echo "Usage: ./build.sh [-h]"
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

go build -o $COMMAND
tar cvf build.tar bin/sfu conf/conf.toml conf/cert.pem conf/key.pem scripts/start.sh scripts/stop.sh
