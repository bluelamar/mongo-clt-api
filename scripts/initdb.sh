#!/bin/bash

docker pull mongo:4.2.2

DBPATH=/var/opt/mongodb
DBCFG=/var/opt/mongocfg

mkdir -p $DBPATH
chmod 755 $DBPATH

docker run -d -p 27017-27019:27017-27019 -v ${DBPATH}:/data/db -v ${DBCFG}:/data/configdb --name mongodb mongo:4.2.2

CONTID=`docker ps | grep "mongo:4.2.2" | awk '{ printf $1 }'`
if [ -z "$CONTID" ]; then
  echo "mongodb not running in docker"
  echo 'You may have to rm the container sometimes - indicated by error in log file "context deadline exceeded"'
  exit 0
fi
docker update --restart=always $CONTID

docker exec -it mongodb mongo

