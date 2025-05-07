#!/bin/bash

# 设置数据库连接参数
DB_USER="postgres"
DB_PASSWORD="postgres"
DB_NAME="nfc_card"

# 确保脚本在出错时停止执行
set -e

echo "开始执行数据库迁移..."

# 按照数字顺序执行迁移文件
for file in $(ls -v migrations/0*.sql migrations/add_cover_key_to_videos.sql 2>/dev/null)
do
  echo "执行 $file..."
  cat "$file" | docker exec -i nfc_card-postgres-1 psql -U $DB_USER -d $DB_NAME
  if [ $? -eq 0 ]; then
    echo "成功执行 $file"
  else
    echo "执行 $file 时出错"
    exit 1
  fi
done

echo "数据库迁移完成" 