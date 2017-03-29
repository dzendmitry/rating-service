FROM mongo:3.4.2

ENV REPLNAME ratingservice

RUN mkdir -p /data/arb

EXPOSE 27017 

CMD mongod --nojournal --dbpath /data/arb --replSet $REPLNAME --logpath /var/log/mongodb/mongod.log
