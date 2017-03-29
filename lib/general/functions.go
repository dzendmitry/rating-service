package general

import (
	"crypto/md5"
	"io"
	"fmt"
	"strings"
	"net/http"
	"errors"
	"io/ioutil"
	"time"
)

func BuildUrl(httpType, host, port, uri, params string) string {
	url := httpType + "://" + host
	if port != "" {
		url += ":" + port
	}
	if url[len(url)-1] != '/' {
		url += "/"
	}
	if len(uri ) > 1 && uri[0] == '/' {
		uri = uri[1:]
	}
	url += uri
	if params != "" {
		url += "?" + params
	}
	return url
}

func GetHash(args ...string) string {
	if len(args) == 0 {
		return ""
	}
	hasher := md5.New()
	io.WriteString(hasher, strings.Join(args, " "))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func GetPage(url string, jsonContenyTypeCheck bool) ([]byte, error) {
	return GetPageWithHeaders(url, jsonContenyTypeCheck, make(map[string]string))
}

func GetPageWithHeaders(url string, jsonContenyTypeCheck bool, headers map[string]string) ([]byte, error) {
	respC := make(chan *HttpResponse, 1)
	go func() {
		req, _ := http.NewRequest("GET", url, nil)
		for k, v := range headers {
			req.Header.Add(k, v)
		}
		res, err := http.DefaultClient.Do(req)
		respC <- &HttpResponse{res, err}
	}()
	var err error
	var body []byte
	select {
	case r := <-respC:
		if r.Err != nil {
			return nil, r.Err
		}
		if r.Res.StatusCode != http.StatusOK {
			return nil, errors.New(fmt.Sprintf("Invalid status code: %s", r.Res.Status))
		}
		if jsonContenyTypeCheck && !strings.Contains(r.Res.Header.Get("Content-Type"), "application/json")  {
			return nil, errors.New(fmt.Sprintf("Invalid content type: %s. Need application/json",
				r.Res.Header.Get("Content-Type")))
		}
		body, err = ioutil.ReadAll(io.LimitReader(r.Res.Body, BODY_BUFFER))
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Reading body %s: %s", url, err))
		}
		if err = r.Res.Body.Close(); err != nil {
			return nil, errors.New(fmt.Sprintf("Closing body %s: %s", url, err))
		}
	case <-time.After(time.Millisecond * ONE_REQUEST_TIMEOUT):
		return nil, errors.New(fmt.Sprintf("HTTP request to %s is timed out", url))
	}
	return body, nil
}

func ValidateRequest(req *http.Request, method string, checkJson bool) ([]byte, error, int) {
	if req.Method != method {
		return nil, errors.New(fmt.Sprintf("Wrong http add requst method: %s", req.Method)), http.StatusMethodNotAllowed
	}
	if checkJson && req.Header.Get("Content-Type") != "application/json" {
		return nil, errors.New(fmt.Sprintf("Wrong Content-Type %s. Need application/json", req.Header.Get("Content-Type"))), http.StatusBadRequest
	}
	body, err := ioutil.ReadAll(io.LimitReader(req.Body, BODY_BUFFER))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error while reading data from request %s: %s", req.RequestURI, err.Error())), http.StatusInternalServerError
	}
	if err = req.Body.Close(); err != nil {
		return nil, errors.New(fmt.Sprintf("Error while closing data request %s: %s", req.RequestURI, err.Error())), http.StatusInternalServerError
	}
	return body, nil, 0
}