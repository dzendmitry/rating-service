FROM golang:1.8.0-alpine

WORKDIR /service

RUN mkdir -p json-schema

ADD ./json-schema json-schema/
ADD auth-service .

ENV MONGO_URL="mongodb://mongodb-master:27017,mongodb-slave:27017/ratingservice?replicaSet=ratingservice"
ENV MONGO_DB=ratingservice
ENV REG_JSON_SCHEMA="file:///service/json-schema/reg.json"
ENV AUTH_JSON_SCHEMA="file:///service/json-schema/auth.json"

EXPOSE 8090

CMD /service/auth-service
