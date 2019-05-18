#!/usr/bin/env bash


function build_key()
{
    echo "****************************************"
    echo 'building key.pem and cert.pem to ../conf'
    echo "****************************************"
    echo -e "\n"
    mv ../conf/key.pem ../conf/key.pem.bak.`date +%Y%m%d%H%M%S` 2>/dev/null
    mv ../conf/cert.pem ../conf/cert.pem.bak.`date +%Y%m%d%H%M%S` 2>/dev/null
    openssl req -newkey rsa:2048 -new -nodes -x509 -days 3650 -keyout ../conf/key.pem -out ../conf/cert.pem
    echo 'build ok'
}

function install_signal_server()
{
    read -r -p "Do you want signal server support Centos/Mac? [c/m]" input
    
    echo "****************************************"
    echo 'downloading signal server, please wait'
    echo "****************************************"
    echo -e "\n"
    case $input in
    [Cc])
        rm centrifugo centrifugo.pid centrifugo_2.1.0_linux_amd64.tar.gz 2> /dev/null
        wget -q https://github.com/centrifugal/centrifugo/releases/download/v2.1.0/centrifugo_2.1.0_linux_amd64.tar.gz
        tar xf centrifugo_2.1.0_linux_amd64.tar.gz
        ;;
    
    [Mm])
        rm centrifugo centrifugo.pid centrifugo_2.1.0_linux_amd64.tar.gz 2> /dev/null
        wget -q https://github.com/centrifugal/centrifugo/releases/download/v2.1.0/centrifugo_2.1.0_darwin_amd64.tar.gz
        tar xf centrifugo_2.1.0_darwin_amd64.tar.gz
        ;;
    
    \
        *)
        echo "Invalid input..."
        exit 1
        ;;
    esac
    echo 'download ok!'
}

install_signal_server
build_key
