FROM golang:1.8.0-alpine

WORKDIR /service

ADD kinopoisk-service .

ENV HTTP_HOST="kinopoisk-service"
ENV HTTP_PORT=8081

EXPOSE 8081

CMD /service/kinopoisk-service
