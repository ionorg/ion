#!/bin/bash
APP_DIR=$(cd `dirname $0`/../; pwd)
ETCD_DIR=$APP_DIR/bin/etcd
mkdir -p $ETCD_DIR/log
cd $APP_DIR/bin

#mv to tmp
function saferm()
{
    for i in $*
    do
        mv $i "/tmp/$name`date +%Y%m%d%H%M%S`" > /dev/null 2>&1
    done
}

saferm etcd*
wget https://github.com/etcd-io/etcd/releases/download/v3.3.13/etcd-v3.3.13-linux-amd64.tar.gz
tar xf etcd-v3.3.13-linux-amd64.tar.gz
mv etcd-v3.3.13-linux-amd64 etcd
cd etcd
openssl genrsa -out $ETCD_DIR/app.rsa 2048
openssl rsa -in $ETCD_DIR/app.rsa -pubout > $ETCD_DIR/app.rsa.pub

