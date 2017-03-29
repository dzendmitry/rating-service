package main

import (
	"net/http"
	"github.com/dzendmitry/logger"
	"github.com/xeipuuv/gojsonschema"
	"github.com/dzendmitry/rating-service/lib/mongo"
	"fmt"
	"github.com/dzendmitry/rating-service/lib/general"
	"os"
)

var (
	mongoUrl       = os.Getenv("MONGO_URL")
	mongoDb        = os.Getenv("MONGO_DB")
	regJsonSchema  = os.Getenv("REG_JSON_SCHEMA")
	authJsonSchema = os.Getenv("AUTH_JSON_SCHEMA")
)

func init() {
	if mongoUrl == "" {
		panic("env MONGO_URL is empty")
	}
	if mongoDb == "" {
		panic("env MONGO_DB is empty")
	}
	if regJsonSchema == "" {
		panic("env REG_JSON_SCHEMA is empty")
	}
	if authJsonSchema == "" {
		panic("env AUTH_JSON_SCHEMA is empty")
	}
}

func main() {
	log := logger.InitFileLogger("AUTH-SERVICE", "")
	defer log.Close()

	if err := mongo.Start(mongoUrl, mongoDb, "", ""); err != nil {
		panic(fmt.Sprintf("Connection to mongo failed: %+v", err))
	}
	defer mongo.Close()

	regSchemaLoader := gojsonschema.NewReferenceLoader(regJsonSchema)
	authSchemaLoader := gojsonschema.NewReferenceLoader(authJsonSchema)
	schemaLoaders := map[string]gojsonschema.JSONLoader{
		"reg": regSchemaLoader,
		"auth": authSchemaLoader,
	}

	h := NewHandlers(general.NewValidator(schemaLoaders, log), log)
	defer h.Close()

	http.HandleFunc(regUrl(), h.regHandler)
	http.HandleFunc(unregUrl(), h.unregHandler)
	http.HandleFunc(authUrl(), h.authHandler)
	http.HandleFunc(exitUrl(), h.exitHandler)
	log.Panicf("%v", http.ListenAndServe(":8090", nil))
}