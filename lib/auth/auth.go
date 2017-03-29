package auth

import (
	"github.com/dzendmitry/logger"
	"errors"
	"fmt"
	"gopkg.in/mgo.v2/bson"
	"strings"
	"github.com/dzendmitry/rating-service/lib/general"
	"time"
	"strconv"
	"net/http"
)

const (
	AUTH_VALIDATE = "auth"
	REG_VALIDATE = "reg"

	COOKIE_EXPIRES = 1209600
	SidKey = "sid"
)

type IAuthDataSource interface {
	Insert(query interface{}) error
	FindOne(query interface{}, result interface{}) error
	Remove(selector interface{}) error
}

type Auth struct {
	Validator *general.Validator
	log logger.ILogger
}

func New(validator *general.Validator) *Auth {
	return &Auth{
		Validator: validator,
		log: logger.InitFileLogger("AUTH", ""),
	}
}

func (r *Auth) Close() {
	r.log.Close()
}

func (a *Auth) Register(users IAuthDataSource, query interface{}) error {
	if err := users.Insert(query); err != nil {
		if strings.Contains(err.Error(), "dup key") {
			return ErrorUserExists{}
		}
	}
	return nil
}

func (a *Auth) Unregister(units IAuthDataSource, users IAuthDataSource, query interface{}) error {
	switch o := query.(type) {
	case *Session:
		if err := units.Remove(bson.M{"uid": o.Uid}); err != nil {
			if err.Error() != "not found" {
				return err
			}
		}
		if err := users.Remove(bson.M{"_id": o.Uid}); err != nil {
			if err.Error() == "not found" {
				return ErrorUsersDoesntExist{}
			}
			return err
		}
	default:
		return errors.New("Invalid data")
	}
	return nil
}

func (a *Auth) Auth(users IAuthDataSource, sessions IAuthDataSource, query interface{}) (string, error) {
	switch o := query.(type) {
	case *AuthData:
		var user RegData
		if err := users.FindOne(bson.M{"name": o.Name}, &user); err != nil {
			if err.Error() == "not found" {
				return "", ErrorInvalidLogin{}
			}
			return "", err
		}
		if user.Name != o.Name {
			return "", ErrorInvalidLogin{}
		}
		if user.Password != general.GetHash(o.Password) {
			return "", ErrorInvalidPassword{}
		}
		sid := general.GetHash(
			user.Name,
			user.Email,
			string(user.Id),
			strconv.FormatInt(time.Now().UnixNano(), 10))
		if err := sessions.Insert(bson.M{
			SidKey: sid,
			"uid": user.Id,
			"created": time.Now(),
			}); err != nil {
			return "", err
		}
		return sid, nil
	default:
		return "", errors.New("Invalid data")
	}
	return "", nil
}

func (a *Auth) Exit(sessions IAuthDataSource, cookie *http.Cookie) (*Session, error) {
	var session Session
	if err := sessions.FindOne(bson.M{SidKey: cookie.Value}, &session); err != nil {
		if err.Error() == "not found" {
			return nil, ErrorNotAuthorized{}
		}
		return nil, err
	}
	if err := sessions.Remove(bson.M{"_id": session.Id}); err != nil {
		return nil, err
	}
	return &session, nil
}

func isAuth(req *http.Request, sessions IAuthDataSource) (bool, *Session, error) {
	cookie, err := req.Cookie(SidKey)
	if err != nil {
		return false, nil, errors.New(fmt.Sprintf("Cookie not round in request: %s", req.RequestURI))
	}
	var session Session
	if err := sessions.FindOne(bson.M{SidKey: cookie.Value}, &session); err != nil {
		return false, nil, ErrorNotAuthorized{}
	}
	return true, &session, nil
}

func Is(req *http.Request, sessions IAuthDataSource, log logger.ILogger) (bool, *Session) {
	a, s, err := isAuth(req, sessions)
	if err != nil {
		log.Warn(err.Error())
	}
	return a, s
}