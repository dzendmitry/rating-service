package main

import (
	"net/http"
	"encoding/json"
	"time"
	"github.com/dzendmitry/rating-service/lib/general"
	"github.com/dzendmitry/rating-service/lib/mongo"
	"github.com/dzendmitry/logger"
	"github.com/dzendmitry/rating-service/lib/auth"
)

type Handlers struct {
	log logger.ILogger
	auth *auth.Auth
}

func NewHandlers(validator *general.Validator, log logger.ILogger) *Handlers {
	return &Handlers{
		log: log,
		auth: auth.New(validator),
	}
}

func (h *Handlers) Close() {
	h.auth.Close()
}

func (h *Handlers) regHandler(w http.ResponseWriter, req *http.Request) {
	body, err, status := general.ValidateRequest(req, http.MethodPost, true)
	if err != nil {
		h.log.Warn(err.Error())
		w.WriteHeader(status)
		return
	}

	err, errs := h.auth.Validator.Validate(body, auth.REG_VALIDATE)
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
	if err != nil {
		h.log.Warnf("%+v", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var regObj auth.RegData
	if err := json.Unmarshal(body, &regObj); err != nil {
		h.log.Warnf("Error while unmarshalling json for request %s: %s", req.RequestURI, err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	regObj.Password = general.GetHash(regObj.Password)

	if err := h.auth.Register(mongo.Users, regObj); err != nil {
		h.log.Warnf("Error during the registration process: %s", err.Error())
		if _, ok := err.(auth.ErrorUserExists); ok {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(err.Error()))
		return
	}
}

func (h *Handlers) unregHandler(w http.ResponseWriter, req *http.Request) {
	cookie, err := req.Cookie(auth.SidKey)
	if err != nil {
		h.log.Warnf("Cookie not round in request: %s", req.RequestURI)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sid, err := h.auth.Exit(mongo.Sessions, cookie)
	if err != nil {
		h.log.Warnf("Error during the exit process: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	if err := h.auth.Unregister(mongo.Units, mongo.Users, sid); err != nil {
		h.log.Warnf("Error during the unregistration process: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name: auth.SidKey,
		Value: "",
		Path: "/",
		MaxAge: -1,
	})
}

func (h *Handlers) authHandler(w http.ResponseWriter, req *http.Request) {
	body, err, status := general.ValidateRequest(req, http.MethodPost, true)
	if err != nil {
		h.log.Warn(err.Error())
		w.WriteHeader(status)
		return
	}

	err, errs := h.auth.Validator.Validate(body, auth.AUTH_VALIDATE)
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
	if err != nil {
		h.log.Warnf("%+v", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var authObj auth.AuthData
	if err := json.Unmarshal(body, &authObj); err != nil {
		h.log.Warnf("Error while unmarshalling json for request %s: %s", req.RequestURI, err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if a, _ := auth.Is(req, mongo.Sessions, h.log); a {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(auth.ErrorAlreadyAuthenticated{}.Error()))
		return
	}

	sid, err := h.auth.Auth(mongo.Users, mongo.Sessions, &authObj)
	if err != nil {
		h.log.Warnf("Error during the auth process: %s", err.Error())
		if _, ok := err.(auth.ErrorInvalidLogin); ok {
			w.WriteHeader(http.StatusBadRequest)
		} else if _, ok := err.(auth.ErrorInvalidPassword); ok {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(err.Error()))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name: auth.SidKey,
		Value: sid,
		Path: "/",
		Expires: time.Now().Add(auth.COOKIE_EXPIRES * time.Second),
	})
}

func (h *Handlers) exitHandler(w http.ResponseWriter, req *http.Request) {
	cookie, err := req.Cookie(auth.SidKey)
	if err != nil {
		h.log.Warnf("Cookie not found in request: %s", req.RequestURI)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if _, err := h.auth.Exit(mongo.Sessions, cookie); err != nil {
		h.log.Warnf("Error during the exit process: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name: auth.SidKey,
		Value: "",
		Path: "/",
		MaxAge: -1,
	})
}