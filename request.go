package httputil

import (
	"errors"
	"net/http"
	"strings"
)

type (
	Request interface {
		Path() Path
		Header() http.Header // embed ReqHeaders into your model for this
		Validate() error
	}
	ReqHeaders struct {
		ReqID string // optional req header
		h     http.Header
	}
)

var (
	ErrInvalidRequest = errors.New("request invalid")
	ErrNewRequestFail = errors.New("http.NewRequest failed")
)

func (r *ReqHeaders) AddHeader(key string, vals ...string) {
	if r.h == nil {
		r.h = make(http.Header)
	}
	for _, val := range vals {
		r.h.Add(key, val)
	}
}
func (r *ReqHeaders) SetHeader(key string, vals ...string) {
	if r.h == nil {
		r.h = make(http.Header)
	}
	for _, val := range vals {
		r.h.Set(key, val)
	}
}
func (r *ReqHeaders) Header() http.Header {
	if r.ReqID != "" {
		r.AddHeader(HeaderReqID, r.ReqID)
	}
	return r.h
}

func reqWithHeaders(req *http.Request, headerMaps ...http.Header) *http.Request {
	for _, headers := range headerMaps {
		for k, values := range headers {
			for _, v := range values {
				req.Header.Add(k, v)
			}
		}
	}
	return req
}
func uriWithBase(uri, baseURL string) string {
	if baseURL != "" && !strings.HasPrefix(uri, baseURL) {
		return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(uri, "/")
	}
	return uri
}
