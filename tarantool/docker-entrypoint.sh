#!/bin/bash
set -e

if [[ -z "${TARANTOOL_WORK_DIR}" ]]; then
    echo "TARANTOOL_WORK_DIR must be defined"
    exit 1
fi

if [[ -z "${TARANTOOL_RUN_DIR}" ]]; then
    echo "TARANTOOL_RUN_DIR was not provided. Setting default: ${TARANTOOL_WORK_DIR}/run"
    export TARANTOOL_RUN_DIR=$TARANTOOL_WORK_DIR/run
fi

mkdir -p $TARANTOOL_WORK_DIR $TARANTOOL_LOG_DIR $TARANTOOL_RUN_DIR

export LD_LIBRARY_PATH=/usr/local/lib64/:$LD_LIBRARY_PATH


chmod 0755 /usr/local/lib64/libmosquitto.so
chmod 0755 /usr/local/lib64/libmosquitto.so.1
chmod 0755 /usr/local/lib64/libmosquitto.so.1.6.2

ln -s /usr/local/lib64/libmosquitto.so.1.6.2 /usr/local/lib/libmosquitto.so
ln -s /usr/local/lib64/libmosquitto.so.1.6.2 /usr/local/lib/libmosquitto.so.1
ln -s /usr/local/lib64/libmosquitto.so.1.6.2 /usr/local/lib/libmosquitto.so.1.6.2

ln -s /usr/local/lib64/libmosquitto.so.1.6.2 /opt/tarantool/.rocks/lib/tarantool/mqtt/libmosquitto.so
ln -s /usr/local/lib64/libmosquitto.so.1.6.2 /opt/tarantool/.rocks/lib/tarantool/mqtt/libmosquitto.so.1
ln -s /usr/local/lib64/libmosquitto.so.1.6.2 /opt/tarantool/.rocks/lib/tarantool/mqtt/libmosquitto.so.1.6.2

echo "/usr/local/lib64/" >> /etc/ld.so.conf
ldconfig -v
export LD_PRELOAD=/usr/local/lib64/libmosquitto.so.1 
exec "$@"
