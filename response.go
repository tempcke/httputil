package httputil

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// DecodeResponse attempts to decode the response body into out
// it will however replace the response body so that it can be read again
// it also returns the body bytes so that you can debug if it did not work as expected
func DecodeResponse(r *http.Response, out any) ([]byte, error) {
	var (
		buf bytes.Buffer
		tr  = io.TeeReader(r.Body, &buf)
	)

	if out == nil {
		out = &json.RawMessage{}
	}

	if err := json.NewDecoder(tr).Decode(out); err != nil {
		if buf.Len() == 0 {
			return nil, errors.New("decode empty body")
		}
		r.Body = io.NopCloser(&buf)
		return buf.Bytes(), err
	}

	r.Body = io.NopCloser(&buf)
	return buf.Bytes(), nil
}

func DecodeResOrErrRes(r *http.Response, goodOut, badOut any) ([]byte, error) {
	if r.StatusCode >= 400 {
		return DecodeResponse(r, badOut)
	}
	return DecodeResponse(r, goodOut)
}
