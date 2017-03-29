package main

import (
	"time"
	"errors"
	"fmt"
	"strconv"

	"github.com/dzendmitry/logger"
	"github.com/dzendmitry/rating-service/lib/general"
	"github.com/leominov/gokinopoisk/search"
)

type Kinopoisk struct {
	log logger.ILogger
}

func NewKinopoisk(log logger.ILogger) *Kinopoisk {
	return &Kinopoisk{
		log: log,
	}
}

func (k *Kinopoisk) FindByName(name string) ([]general.ContentUnit, error) {
	dataC := make(chan *search.Data, 1)
	go func() {
		res, err := search.Query(name)
		if err != nil {
			k.log.Warnf("Getting data error: %+v", err)
			dataC <- nil
			return
		}
		dataC <- res
	}()
	res := make([]general.ContentUnit, 0)
	timer := time.After(general.ONE_REQUEST_TIMEOUT * time.Second)
	select {
	case data := <-dataC:
		if data == nil {
			break
		}
		for _, film := range data.Films {
			unit := general.ContentUnit{
				Title: fmt.Sprintf("%s (%s)\n", film.Title, film.OriginalTitle),
				PicUrl: film.Poster.BaseUrl,
				Url: film.URL,
				Type: general.TYPE_MOVIE,
			}
			if len(film.Years) > 0 {
				unit.Year = strconv.Itoa(film.Years[0])
			}
			res = append(res, unit)
		}
	case <- timer:
		return res, errors.New("Data response is timed out")
	}
	return res, nil
}