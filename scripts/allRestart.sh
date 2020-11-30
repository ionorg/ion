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
$APP_DIR/scripts/allStop.sh
$APP_DIR/scripts/allStart.sh



