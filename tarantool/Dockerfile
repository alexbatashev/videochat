FROM tarantool/tarantool:latest

# Tarantool binary protocol port
EXPOSE 3301

COPY tarantool_init.lua /usr/local/share/tarantool/tarantool_init.lua

CMD ["tarantool", "/usr/local/share/tarantool/tarantool_init.lua"]