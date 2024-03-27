package httputil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	HeaderReqID       = "X-Request-ID"
	HeaderContentType = "Content-Type"
	ApplicationJSON   = "application/json"
	Default429Retry   = 2
)

var (
	defaultClient = &http.Client{Timeout: 60 * time.Second}
)

type (
	Client struct {
		ReqHeaders
		HttpClient   httpClient // *http.Client
		Host         string
		log          sLogger
		RateLimiter  *RateLimiter
		RetriesOn429 int
	}
	httpClient interface { // *http.Client
		Do(req *http.Request) (*http.Response, error)
	}
)

func NewClient() Client {
	return new(Client).
		WithLogger(slog.Default()).
		WithHttpClient(defaultClient).
		WithRateLimiter(NewRateLimiter(MaxAllowedCallsPerSecond)).
		With429Retry(Default429Retry)
}
func (c Client) WithHost(v string) Client              { c.Host = v; return c }
func (c Client) WithLogger(v LevelLogger) Client       { c.log = newLogger(v); return c }
func (c Client) WithHttpClient(v httpClient) Client    { c.HttpClient = v; return c }
func (c Client) WithRateLimiter(v *RateLimiter) Client { c.RateLimiter = v; return c }
func (c Client) With429Retry(v int) Client             { c.RetriesOn429 = v; return c }
func (c Client) WithHeader(h http.Header) Client {
	for k, v := range h {
		c.AddHeader(k, v...)
	}
	return c
}

func (c Client) DoReq(ctx context.Context, method string, r Request, out, errRes any) (*http.Response, error) {
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRequest, err)
	}

	var (
		uri    = r.Path().WithBaseURL(c.Host).String()
		header = r.Header()
		body   = r
	)

	req, err := c.Request(ctx, method, uri, header, body)
	if err != nil {
		return nil, err
	}

	res, err := c.DoAndDecode(req, out, errRes)
	if err != nil {
		return res, err
	}
	return res, nil
}
func (c Client) Request(ctx context.Context, method, uri string, headers http.Header, body any) (*http.Request, error) {
	var reqBody io.Reader
	if body != nil {
		switch v := body.(type) {
		case io.Reader:
			reqBody = v
		default:
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", "jsonEncode(body) failed", err)
			}
			if headers == nil {
				headers = make(http.Header)
			}
			headers.Add(HeaderContentType, ApplicationJSON)
			reqBody = bytes.NewReader(bodyBytes)
		}
	}
	uri = uriWithBase(uri, c.Host)
	req, err := http.NewRequestWithContext(ctx, method, uri, reqBody)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", "http.NewRequest failed", err)
	}

	return reqWithHeaders(req, headers), nil
}
func (c Client) DoAndDecode(req *http.Request, out, errRes any) (*http.Response, error) {
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if _, err = DecodeResOrErrRes(res, out, errRes); err != nil {
		return res, err
	}
	return res, nil
}
func (c Client) Do(req *http.Request) (*http.Response, error) {
	var tries = 0
	res, err := c.do(req)
	for res != nil && res.StatusCode == http.StatusTooManyRequests && tries < c.RetriesOn429 {
		tries++
		res, err = c.do(req)
	}
	return res, err
}

func (c Client) do(req *http.Request) (*http.Response, error) {
	if err := c.RateLimiter.Wait(req.Context()); err != nil {
		return nil, err
	}
	res, err := c.httpClient().Do(reqWithHeaders(req, c.Header()))
	c.logRqRs(req, res, err)
	if res != nil && res.StatusCode == http.StatusTooManyRequests {
		c.RateLimiter.SlowDown()
	}
	return res, err
}
func (c Client) httpClient() httpClient {
	if c.HttpClient != nil {
		return c.HttpClient
	}
	return defaultClient
}
func (c Client) logRqRs(req *http.Request, res *http.Response, err error) {
	msg := fmt.Sprintf("[RQ/RS] %s %s", req.Method, req.URL.Path)
	reqFields := fMap{
		"method": req.Method,
		"path":   req.URL.Path,
		"header": req.Header,
		"query":  req.URL.RawQuery,
	}

	if err != nil {
		c.log.Error(msg, fMap{
			"req":   reqFields,
			"error": err.Error(),
		})
		return
	}

	resFields := fMap{
		"status": res.Status,
		"header": res.Header,
	}
	var out = make(map[string]any)
	if bodyBytes, err := DecodeResponse(res, &out); err == nil {
		if len(bodyBytes) < 1024 {
			resFields["body"] = out
		} else {
			resFields["head"] = string(bodyBytes[:1024])
		}
	}
	fields := fMap{
		"req": reqFields,
		"res": resFields,
	}
	msg += " " + res.Status
	if res.StatusCode >= 400 {
		c.log.Warn(msg, fields)
		return
	}
	c.log.Info(msg, fields)
}
