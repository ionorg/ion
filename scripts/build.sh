#!/bin/bash
APP_DIR=$(cd `dirname $0`/../; pwd)
cd $APP_DIR
EXE=ion
COMMAND=$APP_DIR/bin/$EXE


if [ -f /etc/os-release ]; then
    # freedesktop.org and systemd
    . /etc/os-release
    CPU=`cat /proc/cpuinfo | grep "processor" | wc -l`
    MEM=`free -b|grep "Mem"|awk -F' ' '{print $2}'`
    OS_TYPE=$NAME
    unset NAME
    OS_VER=$VERSION_ID
elif type lsb_release >/dev/null 2>&1; then
    # linuxbase.org
    CPU=`cat /proc/cpuinfo | grep "processor" | wc -l`
    MEM=`free -b|grep "Mem"|awk -F' ' '{print $2}'`
    OS_TYPE=$(lsb_release -si)
    OS_VER=$(lsb_release -sr)
elif [ -f /etc/lsb-release ]; then
    # For some versions of Debian/Ubuntu without lsb_release command
    . /etc/lsb-release
    CPU=`cat /proc/cpuinfo | grep "processor" | wc -l`
    MEM=`free -b|grep "Mem"|awk -F' ' '{print $2}'`
    OS_TYPE=$DISTRIB_ID
    OS_VER=$DISTRIB_RELEASE
elif [ -f /etc/debian_version ]; then
    # Older Debian/Ubuntu/etc.
    CPU=`cat /proc/cpuinfo | grep "processor" | wc -l`
    MEM=`free -b|grep "Mem"|awk -F' ' '{print $2}'`
    OS_TYPE=Debian
    OS_VER=$(cat /etc/debian_version)
elif [ -f /etc/SuSe-release ]; then
    # Older SuSE/etc.
    CPU=`cat /proc/cpuinfo | grep "processor" | wc -l`
    MEM=`free -b|grep "Mem"|awk -F' ' '{print $2}'`
    ...
elif [ -f /etc/redhat-release ]; then
    # Older Red Hat, CentOS, etc.
    CPU=`cat /proc/cpuinfo | grep "processor" | wc -l`
    MEM=`free -b|grep "Mem"|awk -F' ' '{print $2}'`
    ...
else
    # Fall back to uname, e.g. "Linux <version>", also works for BSD, etc.
    OS_TYPE=$(uname -s)
    OS_VER=$(uname -r)
    CPU=`sysctl -n machdep.cpu.thread_count`
    MEM=`sysctl -n hw.memsize`
fi


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

go build -o $COMMAND
tar cvf build.tar bin conf/conf.toml conf/cert.pem conf/key.pem scripts/start.sh scripts/stop.sh
