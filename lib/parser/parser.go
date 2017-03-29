package parser_service

import (
	"github.com/dzendmitry/logger"
	"net/http"
	"encoding/json"
	"github.com/dzendmitry/rating-service/lib/general"
)

const (
	FIND_URL = "find"
)

func FindUri() string {
	return general.BASE_URL_V1 + FIND_URL
}

type IHandler interface {
	FindByName(name string) ([]general.ContentUnit, error)
}

type ParserHandler struct {
	log     logger.ILogger
	md5Hash string
	parser  IHandler
}

func NewParser(md5Hash string, parser IHandler, log logger.ILogger) *ParserHandler {
	return &ParserHandler{
		log: log,
		md5Hash: md5Hash,
		parser: parser,
	}
}

func (h *ParserHandler) FindHandler(w http.ResponseWriter, req *http.Request) {
	var cu []general.ContentUnit
	var err error
	switch req.FormValue("type") {
	case general.FIND_BY_NAME:
		name := req.FormValue("name")
		if name == "" {
			h.log.Warnf("There is no 'name' parameter in find request type: %s", req.FormValue("type"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		cu, err = h.parser.FindByName(name)
	case "":
		h.log.Warn("Empty type in find request")
		w.WriteHeader(http.StatusBadRequest)
		return
	default:
		h.log.Warnf("Wrong type in find request: %s", req.FormValue("type"))
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	if err != nil {
		h.log.Warnf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	cr := general.ContentResp(cu)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cr); err != nil {
		h.log.Warnf("Error while encoding find response: %+v", err.Error())
		w.WriteHeader(http.StatusExpectationFailed)
		return
	}
}
