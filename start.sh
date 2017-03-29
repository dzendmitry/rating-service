#!/bin/sh

for i in {2..12}
do
	sudo ifconfig lo0 alias 172.18.0.$i
done

docker-compose up $1
