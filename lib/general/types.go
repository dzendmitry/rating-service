package general

import (
	"net/http"
	"gopkg.in/mgo.v2/bson"
	"time"
)

const (
	TYPE_MOVIE = "movie"
	TYPE_BOOK = "book"
)

var ParserTypes map[string]bool = map[string]bool{
	TYPE_MOVIE: true,
	TYPE_BOOK : true,
}

type HttpResponse struct {
	Res *http.Response
	Err error
}

type ContentUnit struct {
	Id bson.ObjectId  `json:"id"       bson:"_id,omitempty"`
	Stars int         `json:"stars"    bson:"stars"`
	Comment string    `json:"comment"  bson:"comment"`
	Edited time.Time  `json:"edited"   bson:"edited"`
	Created time.Time `json:"created"  bson:"created"`
	Sid string        `json:"-"        bson:"sid"`
	Uid bson.ObjectId `json:"-"        bson:"uid"`
	Type string       `json:"type"     bson:"type"`
	Url string        `json:"url"      bson:"url"`
	Title string      `json:"title"    bson:"title"`
	PicUrl string     `json:"pic_url"  bson:"picurl"`
	Desc string       `json:"desc"     bson:"desc"`
	Year string       `json:"year"     bson:"year"`
	Author string     `json:"author"   bson:"author"`
	Isbn string       `json:"isbn"     bson:"isbn"`
}

type ContentResp []ContentUnit

type ContentRespErr struct {
	ContentResp
	Err error
}