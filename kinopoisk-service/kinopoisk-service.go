package main

import (
	"net/http"

	"github.com/dzendmitry/rating-service/lib/general"
	"github.com/dzendmitry/rating-service/lib/udp"
	"github.com/dzendmitry/logger"
	"github.com/dzendmitry/rating-service/lib/parser"
	"os"
)

var (
	httpHost = os.Getenv("HTTP_HOST")
	httpPort = os.Getenv("HTTP_PORT")
)

func init() {
	if httpHost == "" {
		panic("env HTTP_HOST is empty")
	}
	if httpPort == "" {
		panic("env HTTP_PORT is empty")
	}
}

func main() {
	log := logger.InitFileLogger("KINOPOISK-SERVICE", "")
	defer log.Close()

	md5Hash := general.GetHash("kinopoisk-service", general.TYPE_MOVIE)
	if md5Hash == "" {
		log.Panic("md5 hash is empty")
		panic("md5 hash is empty")
	}

	udp.NewParsersSender("224.0.0.1", "9999", udp.ParserMessage{
		Name:       "kinopoisk-service",
		ParserType: general.TYPE_MOVIE,
		Hash:       md5Hash,
		HttpHost:   httpHost,
		HttpPort:   httpPort,
	}, log).Send()

	h := parser_service.NewParser(md5Hash, NewKinopoisk(log), log)

	http.HandleFunc(parser_service.FindUri(), h.FindHandler)
	log.Panicf("%+v", http.ListenAndServe(":" + httpPort, nil))
}
