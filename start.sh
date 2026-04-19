#!/bin/sh

# 脚本中任何命令失败，脚本立即退出
set -e

echo "run db migration"
/app/migrate -path /app/db/migration -database "$DB_SOURCE" -verbose up

echo "start the app"
exec "$@" # 执行 CMD 传进来的指令 (/app/main)