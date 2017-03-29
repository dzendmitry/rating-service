package auth

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

type RegData struct {
	Id bson.ObjectId `bson:"_id,omitempty"`
	Name string `json:"name" bson:"name"`
	Password string `json:"password" bson:"password"`
	Email string `json:"email" bson:"email"`
}

type AuthData struct {
	Name string `json:"name" bson:"name"`
	Password string `json:"password" bson:"password"`
}

type Session struct {
	Id bson.ObjectId `bson:"_id"`
	Sid string `bson:"sid"`
	Uid bson.ObjectId `bson:"uid"`
	Name string `bson:"name"`
	Email string `bson:"email"`
	Created time.Time `bson:"created"`
}