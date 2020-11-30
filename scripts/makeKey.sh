#!/bin/bash
set -eux

APP_DIR=$(cd `dirname $0`/../; pwd)
cd $APP_DIR
PATH=$APP_DIR/configs

echo "****************************************"
echo "building key.pem and cert.pem to $PATH"
echo "****************************************"
echo -e "\n"
mv $PATH/key.pem $PATH/key.pem.bak.`/bin/date +%Y%m%d%H%M%S` 2>/dev/null
mv $PATH/cert.pem $PATH/cert.pem.bak.`/bin/date +%Y%m%d%H%M%S` 2>/dev/null
/usr/bin/openssl req -newkey rsa:2048 -new -nodes -x509 -days 3650 -keyout $PATH/key.pem -out $PATH/cert.pem
echo 'build ok'

