#!/bin/bash

set -x
MONGO_LOG="/var/log/mongodb/mongod.log"
MONGO="/usr/bin/mongo"
MONGOD="/usr/bin/mongod"
 
function checkSlaveStatus() {
	SLAVE=$1
	$MONGO --host $SLAVE --eval db
	while [ "$?" -ne 0 ]
	do
		echo "Waiting for slave to come up..."
		sleep 15
		$MONGO --host $SLAVE --eval db
	done
}
 
if [ "$ROLE" == "master" ]
then
	$MONGOD --fork --replSet $REPLNAME --logpath $MONGO_LOG &
	if [ "$INITIALIZE" == "yes" ]
	then
		sleep 2
		$MONGO --eval "rs.initiate({_id : \"${REPLNAME}\", members: [ { _id : 0, host : \"mongodb-master:27017\" } ]})"
		checkSlaveStatus mongodb-slave
		$MONGO --eval "rs.add(\"mongodb-slave:27017\")"
		checkSlaveStatus mongodb-arbiter
		$MONGO --eval "rs.addArb(\"mongodb-arbiter:27017\")"
		while read cmd
		do
			$MONGO $BASE --eval "$cmd"
		done < /data/mongo-base
	fi
	tailf /dev/null
else
	sleep 10
	$MONGOD --replSet $REPLNAME --logpath $MONGO_LOG
fi
