#!/bin/bash
set -eux

export ROOT_DIR=$(cd `dirname $0`/../; pwd)

export ETCD_VER=v3.3.18
export NATS_VER=v2.1.4
export REDIS_VER=5.0.7
export TIDB_VER=3.0.9

case $(uname | tr '[:upper:]' '[:lower:]') in
  linux*)
    export OS=linux
    ;;
  darwin*)
    export OS=darwin
    ;;
  msys*)
    export OS=windows
    ;;
  *)
    export OS=notset
    ;;
esac

export SRV_DIR=${ROOT_DIR}/deps/${OS}
export ETCD_DIR=${SRV_DIR}/etcd-server
export NATS_DIR=${SRV_DIR}/nats-server
export REDIS_DIR=${SRV_DIR}/redis-server
export TIDB_DIR=${SRV_DIR}/tidb-server

function install_etcd_server() {
    GITHUB_URL=https://github.com/etcd-io/etcd/releases/download
    DOWNLOAD_URL=${GITHUB_URL}
    SUFFIX=.zip
    if [ ${OS} == "linux" ]; then
        SUFFIX=.tar.gz
    fi
    FILE=etcd-${ETCD_VER}-${OS}-amd64${SUFFIX}

    rm -f /tmp/${FILE}
    curl -L ${DOWNLOAD_URL}/${ETCD_VER}/${FILE} -o /tmp/${FILE}

    if [ ${OS} == "linux" ]; then
        tar zxvf /tmp/${FILE} -C /tmp && rm -f /tmp/${FILE}
    else
        unzip /tmp/${FILE} -d /tmp && rm -f /tmp/${FILE}
    fi

    mv /tmp/etcd-${ETCD_VER}-${OS}-amd64/* ${ETCD_DIR} && rm -rf /tmp/etcd-${ETCD_VER}-${OS}-amd64
    ${ETCD_DIR}/etcd --version
    ${ETCD_DIR}/etcdctl --version
}

function install_nats_server() {
    GITHUB_URL=https://github.com/nats-io/nats-server/releases/download
    DOWNLOAD_URL=${GITHUB_URL}

    rm -f /tmp/nats-server-${NATS_VER}-${OS}-amd64.zip
    curl -L ${DOWNLOAD_URL}/${NATS_VER}/nats-server-${NATS_VER}-${OS}-amd64.zip -o /tmp/nats-server-${NATS_VER}-${OS}-amd64.zip
    unzip /tmp/nats-server-${NATS_VER}-${OS}-amd64.zip -d /tmp && rm -f /tmp/nats-server-${NATS_VER}-${OS}-amd64.zip
    mv /tmp/nats-server-${NATS_VER}-${OS}-amd64/* ${NATS_DIR} && rm -rf /tmp/nats-server-${NATS_VER}-${OS}-amd64
    ${NATS_DIR}/nats-server --version
}

function install_redis_server() {
    DOWNLOAD_URL=http://download.redis.io/releases/redis-${REDIS_VER}.tar.gz

    rm -f /tmp/redis-${REDIS_VER}.tar.gz
    curl -L ${DOWNLOAD_URL} -o /tmp/redis-${REDIS_VER}.tar.gz
    tar zxvf /tmp/redis-${REDIS_VER}.tar.gz -C /tmp && rm -f /tmp/redis-${REDIS_VER}.tar.gz
    cd /tmp/redis-${REDIS_VER} && make
    cp -rf /tmp/redis-${REDIS_VER}/src/redis-{server,cli} ${REDIS_DIR} && rm -f /tmp/redis-${REDIS_VER}
    ${REDIS_DIR}/redis-server --version
    ${REDIS_DIR}/redis-cli --version
}

function install_tidb_server() {
    DOWNLOAD_URL=https://github.com/pingcap/tidb/archive/v${TIDB_VER}.tar.gz

    rm -f /tmp/tidb-${TIDB_VER}.tar.gz
    curl -L ${DOWNLOAD_URL} -o /tmp/tidb-${TIDB_VER}.tar.gz
    tar zxvf /tmp/tidb-${TIDB_VER}.tar.gz -C /tmp && rm -f /tmp/tidb-${TIDB_VER}.tar.gz
    cd /tmp/tidb-${TIDB_VER} && make
    cp -rf /tmp/tidb-${TIDB_VER}/bin/tidb-server ${TIDB_DIR} && rm -rf /tmp/tidb-${TIDB_VER}
    ${TIDB_DIR}/tidb-server -V
}

echo "Install run dependencies."

if [ ! -f ${ETCD_DIR}/etcd ]; then
    echo "Install ETCD for ${OS}."
    mkdir -p $SRV_DIR/etcd-server
    install_etcd_server
else
    echo "ECTD for ${OS} installed."
fi

if [ ! -f ${NATS_DIR}/nats-server ]; then
    echo "Install NATS-Server for ${OS}."
    mkdir -p $SRV_DIR/nats-server
    install_nats_server
else
    echo "NATS-Server for ${OS} installed."
fi

if [ ! -f ${REDIS_DIR}/redis-server ]; then
    echo "Install Redis-Server for ${OS}."
    mkdir -p $SRV_DIR/redis-server
    install_redis_server
else
    echo "Redis-Server for ${OS} installed."
fi

if [ ! -f ${TIDB_DIR}/tidb-server ]; then
    echo "Install TiDB-Server for ${OS}."
    mkdir -p $SRV_DIR/tidb-server
    install_tidb_server
else
    echo "TiDB-Server for ${OS} installed."
fi

echo "Done"