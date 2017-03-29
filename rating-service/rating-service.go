package main

import (
	"net/http"

	"github.com/dzendmitry/rating-service/lib/udp"
	"github.com/dzendmitry/logger"
	"github.com/dzendmitry/rating-service/lib/redis"
	"github.com/dzendmitry/rating-service/lib/mongo"
	"fmt"
	"github.com/xeipuuv/gojsonschema"
	"github.com/dzendmitry/rating-service/lib/general"
	"os"
)

var (
	mongoUrl       = os.Getenv("MONGO_URL")
	mongoDb        = os.Getenv("MONGO_DB")
	ucpJsonSchema  = os.Getenv("UCP_JSON_SCHEMA")
	ifis           = os.Getenv("INTERFACE")
	sentinel1       = os.Getenv("REDIS_SENTINEL_1")
	sentinel2       = os.Getenv("REDIS_SENTINEL_2")
	sentinel3       = os.Getenv("REDIS_SENTINEL_3")
)

func init() {
	if mongoUrl == "" {
		panic("env MONGO_URL is empty")
	}
	if mongoDb == "" {
		panic("env MONGO_DB is empty")
	}
	if ucpJsonSchema == "" {
		panic("env REG_JSON_SCHEMA is empty")
	}
	if ifis == "" {
		panic("env INTERFACE is empty")
	}
	if sentinel1 == "" {
		panic("env REDIS_SENTINEL_1 is empty")
	}
	if sentinel2 == "" {
		panic("env REDIS_SENTINEL_2 is empty")
	}
	if sentinel3 == "" {
		panic("env REDIS_SENTINEL_3 is empty")
	}
}

func main() {
	log := logger.InitFileLogger("RATING-SERVICE", "")
	defer log.Close()

	if err := mongo.Start(mongoUrl, mongoDb, "", ""); err != nil {
		panic(fmt.Sprintf("Connection to mongo failed: %+v", err))
	}
	defer mongo.Close()

	if err := redis.Start([]string{sentinel1, sentinel2, sentinel3}, []string{redis.CACHE_EVICT}); err != nil {
		log.Warnf("Redis starting error: %+v", err)
	} else {
		defer redis.Close()
	}

	pl := udp.NewParsersListener("224.0.0.1", "9999", ifis, log)
	pl.Listen()

	ucp := gojsonschema.NewReferenceLoader(ucpJsonSchema)
	schemaLoaders := map[string]gojsonschema.JSONLoader{
		CONTENT_USER_PART_VALIDATE: ucp,
	}

	h := NewHandlers(pl.GetTypeC(), general.NewValidator(schemaLoaders, log), log)

	http.HandleFunc(getMoviesUrl(), h.getContent)
	http.HandleFunc(getBooksUrl(), h.getContent)
	http.HandleFunc(getAllContentUrl(), h.getContent)

	http.HandleFunc(addMovieUrl(), h.addHandler)
	http.HandleFunc(addBookUrl(), h.addHandler)

	http.HandleFunc(findMovieUrl(), h.findHandler)
	http.HandleFunc(findBookUrl(), h.findHandler)

	http.HandleFunc(editMovieUrl(), h.editHandler)
	http.HandleFunc(editBookUrl(), h.editHandler)

	http.HandleFunc(removeMovieUrl(), h.removeHandler)
	http.HandleFunc(removeBookUrl(), h.removeHandler)

	log.Panicf("%v", http.ListenAndServe(":8080", nil))
}