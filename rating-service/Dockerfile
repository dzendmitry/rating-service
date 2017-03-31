FROM golang:1.8.0-alpine

WORKDIR /service

RUN mkdir -p json-schema

ADD ./json-schema json-schema/
ADD rating-service .

ENV MONGO_URL="mongodb://mongodb-master:27017,mongodb-slave:27017/ratingservice?replicaSet=ratingservice"
ENV MONGO_DB=ratingservice
ENV UCP_JSON_SCHEMA="file:///service/json-schema/user-content-part.json"
ENV INTERFACE=eth0
ENV REDIS_SENTINEL_1="redis-sentinel:26379"
ENV REDIS_SENTINEL_2="redis-sentinel-2:26379"
ENV REDIS_SENTINEL_3="redis-sentinel-3:26379"

EXPOSE 8080

CMD /service/rating-service
