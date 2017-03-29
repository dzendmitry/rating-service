package main

import (
	"net/http"
	"strings"
	"fmt"
	"time"
	"errors"
	"encoding/json"

	"gopkg.in/mgo.v2/bson"
	"github.com/dzendmitry/rating-service/lib/udp"
	"github.com/dzendmitry/logger"
	"github.com/dzendmitry/rating-service/lib/redis"
	"github.com/dzendmitry/rating-service/lib/parser"
	"github.com/dzendmitry/rating-service/lib/general"
	"github.com/dzendmitry/rating-service/lib/auth"
	"github.com/dzendmitry/rating-service/lib/mongo"
)

const (
	CONTENT_USER_PART_VALIDATE = "content-user-part"
)

type Handlers struct {
	plTypeC chan udp.GetParsersCmd
	validator *general.Validator
	log logger.ILogger
}

func NewHandlers(plTypeC chan udp.GetParsersCmd, validator *general.Validator, log logger.ILogger) *Handlers {
	return &Handlers{
		plTypeC: plTypeC,
		validator: validator,
		log: log,
	}
}

func (h *Handlers) request(url string, contRespErrC chan *general.ContentRespErr) {
	body, err := general.GetPage(url, true)
	if err != nil {
		h.log.Warn(err.Error())
		contRespErrC <- &general.ContentRespErr{
			Err: err,
		}
		return
	}
	var cre general.ContentRespErr
	if err := json.Unmarshal(body, &cre.ContentResp); err != nil {
		h.log.Warnf("Unmarshalling parser answer from %s: %s", url, err.Error())
		contRespErrC <- &general.ContentRespErr{
			Err: errors.New(fmt.Sprintf("Unmarshalling parser answer from %s: %s", url, err.Error())),
		}
		return
	}
	contRespErrC <- &cre
}

func (h *Handlers) processParserResps(parsersResps []general.ContentResp, uid bson.ObjectId, sid string) []general.ContentResp {
	for i := range parsersResps {
		for j := 0; j < len(parsersResps[i]); j++ {
			parsersResps[i][j].Id = bson.NewObjectId()
			parsersResps[i][j].Created = time.Now()
			parsersResps[i][j].Edited = parsersResps[i][j].Created
			parsersResps[i][j].Sid = sid
			parsersResps[i][j].Uid = uid
			if err := mongo.Answers.Insert(parsersResps[i][j]); err != nil {
				h.log.Warnf("Database inserting error: %s", err.Error())
				parsersResps[i] = append(parsersResps[i][:j], parsersResps[i][j+1:]...)
				j -= 1
			}
		}
	}
	for i := 0; i < len(parsersResps); i++ {
		if len(parsersResps[i]) == 0 {
			parsersResps = append(parsersResps[:i], parsersResps[i+1:]...)
			i -= 1
		}
	}
	return parsersResps
}

func (h *Handlers) processParsersRequests(parsers []udp.ParserUnit, uri, params string) []general.ContentResp {
	respC := make(chan *general.ContentRespErr, len(parsers))
	for _, parser := range parsers {
		parserUrl := general.BuildUrl("http", parser.HttpHost, parser.HttpPort, uri, params)
		go h.request(parserUrl, respC)
	}

	contentResponses := make([]general.ContentResp, 0, len(parsers))
	timer := time.NewTimer(time.Duration(len(parsers)) * general.ONE_PARSER_REQUEST_TIMEOUT * time.Second)
L:
	for i := 0; i < len(parsers); i++ {
		select {
		case r := <- respC:
			if r.Err != nil {
				continue
			}
			contentResponses = append(contentResponses, r.ContentResp)
		case <- timer.C:
			h.log.Warnf("Requests to parsers %#v are timed out", parsers)
			break L
		}
	}
	return contentResponses
}

func (h *Handlers) findHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.log.Warnf("Wrong http find requst method: %s", req.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	a, session := auth.Is(req, mongo.Sessions, h.log)
	if !a {
		h.log.Warnf("Non authorized request: %+v from ip: %+v", req.RequestURI, req.RemoteAddr)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	var reqType string
	switch {
	case strings.HasPrefix(req.RequestURI, findMovieUrl()):
		reqType = general.TYPE_MOVIE
	case strings.HasPrefix(req.RequestURI, findBookUrl()):
		reqType = general.TYPE_BOOK
	default:
		h.log.Warnf("Wrong type in find request: %s", req.RequestURI)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	cacheData, err := redis.GetSentiel(redis.CACHE_EVICT, req.RequestURI)
	if err != nil {
		h.log.Warnf("Error during redis evict cache GET request: %+v", err.Error())
	} else {
		var parsersResps []general.ContentResp
		if err := json.Unmarshal(cacheData, &parsersResps); err != nil {
			h.log.Warnf("Error while unmarshalling data from cache: %+v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		parsersResps = h.processParserResps(parsersResps, session.Uid, session.Sid)
		if len(parsersResps) == 0 {
			h.log.Warnf("There is no data after relocation to mongo: %s", req.RequestURI)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(parsersResps); err != nil {
			h.log.Warnf("Error while json encoding cache response: %+v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		return
	}

	parsersC := make(chan []udp.ParserUnit)
	h.plTypeC <- udp.GetParsersCmd{
		Type: reqType,
		C: parsersC,
	}
	parsers := <- parsersC
	if len(parsers) == 0 {
		h.log.Warnf("There are no parsers for type: %s", reqType)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch req.FormValue("type") {
	case general.FIND_BY_NAME:
		if req.FormValue("name") == "" {
			h.log.Warnf("There is no 'name' parameter in find request type: %s", reqType)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	case "":
		h.log.Warn("Empty type in find request")
		w.WriteHeader(http.StatusBadRequest)
		return
	default:
		h.log.Warnf("Wrong type in find request: %s", req.FormValue("type"))
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	parsersResps := h.processParsersRequests(parsers, parser_service.FindUri(), req.Form.Encode())
	if len(parsersResps) == 0 {
		h.log.Warnf("There is no data to find request: %s", req.RequestURI)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	parsersResps = h.processParserResps(parsersResps, session.Uid, session.Sid)
	if len(parsersResps) == 0 {
		h.log.Warnf("There is no data after relocation to mongo: %s", req.RequestURI)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	data, err := json.Marshal(parsersResps)
	if err != nil {
		h.log.Warnf("Error while marshalling content responses: %+v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	go func() {
		if err := redis.SetExSentiel(redis.CACHE_EVICT, req.RequestURI, data, redis.CACHE_EVICT_EX); err != nil {
			h.log.Fatalf("Error writing to %s cache: %+v", redis.CACHE_EVICT, err.Error())
		}
	}()
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(data); err != nil {
		h.log.Warnf("Error while writing find response: %+v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) addHandler(w http.ResponseWriter, req *http.Request) {
	a, session := auth.Is(req, mongo.Sessions, h.log)
	if !a {
		h.log.Warnf("Non authorized request: %+v", req.RequestURI)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	body, err, status := general.ValidateRequest(req, http.MethodPost, true)
	if err != nil {
		h.log.Warn(err.Error())
		w.WriteHeader(status)
		return
	}

	err, errs := h.validator.Validate(body, CONTENT_USER_PART_VALIDATE)
	if errs != nil {
		if err != nil {
			h.log.Warnf("%+v", err.Error())
		}
		h.log.Warnf("The document is not valid. see errors :\n")
		h.log.Warnf("%+v", errs)
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(errs); err != nil {
			h.log.Warn("Error while encoding validation errors: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/plain")
		}
		return
	}

	var cont general.ContentUnit
	if err := json.Unmarshal(body, &cont); err != nil {
		h.log.Warnf("Error during unmarshall: %+v", err.Error())
		w.WriteHeader(http.StatusExpectationFailed)
		return
	}
	cont.Edited = time.Now()

	var cu general.ContentUnit
	if err := mongo.Answers.FindOne(bson.M{"_id": cont.Id, auth.SidKey: session.Sid, "uid": session.Uid}, &cu); err != nil {
		h.log.Warnf("Threr is no such result in cache. Maybe it's too late: %+v", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	cu.Edited = time.Now()
	cu.Stars = cont.Stars
	cu.Comment = cont.Comment
	cu.Uid = session.Uid

	if err := mongo.Units.Insert(cu); err != nil {
		h.log.Warnf("Error updateing users content: %+v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) getContent(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.log.Warnf("Wrong http find requst method: %s", req.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	a, session := auth.Is(req, mongo.Sessions, h.log)
	if !a {
		h.log.Warnf("Non authorized request: %+v", req.RequestURI)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	var cu []general.ContentUnit
	switch {
	case strings.HasPrefix(req.RequestURI, getMoviesUrl()):
		if err := mongo.Units.FindAll(bson.M{"uid": session.Uid, "type": general.TYPE_MOVIE}, &cu); err != nil {
			h.log.Warnf("Error getting data from mongo req %s: %s", req.RequestURI, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	case strings.HasPrefix(req.RequestURI, getBooksUrl()):
		if err := mongo.Units.FindAll(bson.M{"uid": session.Uid, "type": general.TYPE_BOOK}, &cu); err != nil {
			h.log.Warnf("Error getting data from mongo req %s: %s", req.RequestURI, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	case strings.HasPrefix(req.RequestURI, getAllContentUrl()):
		if err := mongo.Units.FindAll(bson.M{"uid": session.Uid}, &cu); err != nil {
			h.log.Warnf("Error getting data from mongo req %s: %s", req.RequestURI, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		h.log.Warnf("Unknown request type: %s", req.RequestURI)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cu); err != nil {
		h.log.Warn("Error while encoding content units: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "text/plain")
	}
}

func (h *Handlers) editHandler(w http.ResponseWriter, req *http.Request) {
	a, session := auth.Is(req, mongo.Sessions, h.log)
	if !a {
		h.log.Warnf("Non authorized request: %+v", req.RequestURI)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	body, err, status := general.ValidateRequest(req, http.MethodPost, true)
	if err != nil {
		h.log.Warn(err.Error())
		w.WriteHeader(status)
		return
	}

	err, errs := h.validator.Validate(body, CONTENT_USER_PART_VALIDATE)
	if errs != nil {
		if err != nil {
			h.log.Warnf("%+v", err.Error())
		}
		h.log.Warnf("The document is not valid. see errors :\n")
		h.log.Warnf("%+v", errs)
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(errs); err != nil {
			h.log.Warn("Error while encoding validation errors: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/plain")
		}
		return
	}

	var cont general.ContentUnit
	if err := json.Unmarshal(body, &cont); err != nil {
		h.log.Warnf("Error during unmarshall: %+v", err.Error())
		w.WriteHeader(http.StatusExpectationFailed)
		return
	}

	var cu general.ContentUnit
	if err := mongo.Answers.FindOne(bson.M{"_id": cont.Id, auth.SidKey: session.Sid}, &cu); err != nil {
		h.log.Warnf("Threr is no such result in cache. Maybe it's too late: %+v", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	cu.Edited = time.Now()
	cu.Stars = cont.Stars
	cu.Comment = cont.Comment

	if err := mongo.Units.Update(bson.M{"_id": cont.Id, "uid": session.Uid}, cu); err != nil {
		h.log.Warnf("Error updateing users content: %+v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) removeHandler(w http.ResponseWriter, req *http.Request) {
	a, session := auth.Is(req, mongo.Sessions, h.log)
	if !a {
		h.log.Warnf("Non authorized request: %+v", req.RequestURI)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	body, err, status := general.ValidateRequest(req, http.MethodPost, true)
	if err != nil {
		h.log.Warn(err.Error())
		w.WriteHeader(status)
		return
	}

	err, errs := h.validator.Validate(body, CONTENT_USER_PART_VALIDATE)
	if errs != nil {
		if err != nil {
			h.log.Warnf("%+v", err.Error())
		}
		h.log.Warnf("The document is not valid. see errors :\n")
		h.log.Warnf("%+v", errs)
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(errs); err != nil {
			h.log.Warn("Error while encoding validation errors: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/plain")
		}
		return
	}

	var cont general.ContentUnit
	if err := json.Unmarshal(body, &cont); err != nil {
		h.log.Warnf("Error during unmarshall: %+v", err.Error())
		w.WriteHeader(http.StatusExpectationFailed)
		return
	}

	if err := mongo.Units.Remove(bson.M{"_id": cont.Id, "uid": session.Uid}); err != nil {
		h.log.Warnf("Error during removing: %+v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}