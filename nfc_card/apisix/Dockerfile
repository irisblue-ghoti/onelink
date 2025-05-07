FROM apache/apisix:3.7.0-debian

USER root
RUN apt-get update && apt-get install -y curl

# 复制配置文件
COPY config-standalone.yaml /usr/local/apisix/conf/config.yaml
COPY plugins /usr/local/apisix/plugins
COPY routes /usr/local/apisix/routes

# 初始化APISIX
RUN cd /usr/local/apisix && \
    /usr/local/openresty/luajit/bin/luajit ./apisix/cli/apisix.lua init -c conf/config.yaml

# 设置启动命令
CMD ["sh", "-c", "cd /usr/local/apisix && /usr/local/openresty/bin/openresty -p /usr/local/apisix -g 'daemon off;'"] 