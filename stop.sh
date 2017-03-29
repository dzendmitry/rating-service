#!/bin/sh

docker-compose stop

for id in `docker ps -a | awk '{print $1}' | grep -v CONTAINER`
do
	docker rm $id
done

for i in {2..9}
do
	sudo ifconfig lo0 -alias 172.18.0.$i
done
