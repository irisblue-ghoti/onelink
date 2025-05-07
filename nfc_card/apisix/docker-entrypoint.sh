#!/bin/bash

# 进入APISIX目录
cd /usr/local/apisix

# 初始化APISIX
echo "初始化APISIX..."
/usr/local/openresty/luajit/bin/luajit ./apisix/cli/apisix.lua init
/usr/local/openresty/luajit/bin/luajit ./apisix/cli/apisix.lua init_etcd

# 启动APISIX
echo "启动APISIX..."
/usr/local/openresty/bin/openresty -p /usr/local/apisix -g "daemon off;" 