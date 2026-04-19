#!/bin/sh

# 使用方式: ./wait-for.sh host:port [-t timeout] [-- command args]

set -e

host="$1"
shift

until nc -z "$host" 2>/dev/null; do
  >&2 echo "Postgres is unavailable - sleeping"
  sleep 1
done

>&2 echo "Postgres is up - executing command"
exec "$@"