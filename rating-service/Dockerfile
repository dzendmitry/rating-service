FROM golang:1.8.0-alpine

WORKDIR /service

RUN mkdir -p json-schema

ADD ./json-schema json-schema/
ADD rating-service .

ENV MONGO_URL="mongodb://mongodb-master:27017,mongodb-slave:27017/ratingservice?replicaSet=ratingservice"
ENV MONGO_DB=ratingservice
ENV UCP_JSON_SCHEMA="file:///service/json-schema/user-content-part.json"
ENV INTERFACE=eth0
ENV REDIS_SENTINEL_1="172.18.0.4:26379"
ENV REDIS_SENTINEL_2="172.18.0.5:26379"
ENV REDIS_SENTINEL_3="172.18.0.6:26379"

EXPOSE 8080

CMD /service/rating-service
