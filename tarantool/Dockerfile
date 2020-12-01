FROM alexbatashev/tarantool:latest

# Tarantool binary protocol port
EXPOSE 3301

ENV TARANTOOL_WORK_DIR=/var/lib/tarantool \
    TARANTOOL_RUN_DIR=/var/run \
    TARANTOOL_MEMTX_MEMORY=134217728

WORKDIR /opt/tarantool
RUN yum install -y gcc make cmake git unzip

COPY app ./
RUN tarantoolctl rocks make app-scm-1.rockspec

# COPY tarantool_init.lua /usr/local/share/tarantool/tarantool_init.lua

# CMD ["tarantool", "/usr/local/share/tarantool/tarantool_init.lua"]

COPY ./docker-entrypoint.sh .

ENTRYPOINT ["/opt/tarantool/docker-entrypoint.sh"]

CMD ["tarantool", "/opt/tarantool/init.lua"]
