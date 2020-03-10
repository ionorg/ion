#!/bin/bash
APP_DIR=$(cd `dirname $0`/../; pwd)
OS_TYPE=""
. $APP_DIR/scripts/common
cd $APP_DIR
EXE1=biz
EXE2=islb

COMMAND1=$APP_DIR/bin/$EXE1
COMMAND2=$APP_DIR/bin/$EXE2


help()
{
    echo ""
    echo "build script"
    echo "Usage: ./build.sh [-h]"
    echo ""
}

while getopts "o:h" arg
do
    case $arg in
        h)
            help;
            exit 0
            ;;
        o)
            OS_TYPE=$OPTARG
            ;;
        ?)
            echo "No argument needed. Ignore them all!"
            ;;
    esac
done

if [[ "$OS_TYPE" == "Darwin" || "$OS_TYPE" == "mac" || "$OS_TYPE" == "darwin" ]];then
    export CGO_ENABLED=1
    export GOOS=darwin
fi

if [[ "$OS_TYPE" == "Ubuntu" || "$OS_TYPE" =~ "CentOS" || "$OS_TYPE" == "ubuntu" || "$OS_TYPE" =~ "centos" || "$OS_TYPE" =~ "linux" || "$OS_TYPE" =~ "Linux" ]];then
    export GOOS=linux
fi

echo "-------------build ion----------"
echo "go build -o $COMMAND1"
cd $APP_DIR/cmd/$EXE1
go build -o $COMMAND1

echo "-------------build islb----------"
echo "go build -o $COMMAND2"
cd $APP_DIR/cmd/$EXE2
go build -o $COMMAND2

cd $APP_DIR
echo "------------tar biz-----------"
tar cvzf biz.tar.gz bin/biz configs/biz.toml configs/cert.pem configs/key.pem scripts/bizStart.sh scripts/bizStop.sh

echo "------------tar islb-----------"
tar cvzf islb.tar.gz bin/islb configs/islb.toml configs/cert.pem configs/key.pem scripts/islbStart.sh scripts/islbStop.sh
