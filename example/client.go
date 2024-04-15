package example

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/tempcke/httputil"
)

const (
	headerAPIKey    = "X-API-Key"
	headerAPISecret = "X-API-Secret"
)

type Client struct {
	client httputil.Client
	logger httputil.LevelLogger
}

func NewClient(baseURL string) Client {
	return new(Client).
		withBase(httputil.NewClient()).
		WithBaseURL(baseURL).
		WithLogger(slog.Default())
}
func (c Client) withBase(b httputil.Client) Client { c.client = b; return c }
func (c Client) WithBaseURL(host string) Client    { return c.withBase(c.client.WithHost(host)) }
func (c Client) With429Retry(retries int) Client   { return c.withBase(c.client.With429Retry(retries)) }
func (c Client) WithRateLimiter(rl *httputil.RateLimiter) Client {
	return c.withBase(c.client.WithRateLimiter(rl))
}
func (c Client) WithLogger(ll httputil.LevelLogger) Client {
	c.logger = ll
	return c.withBase(c.client.WithLogger(ll))
}
func (c Client) WithSetHeader(k string, vals ...string) Client {
	return c.withBase(c.client.WithSetHeader(k, vals...))
}
func (c Client) WithCredentials(key, secret string) Client {
	return c.
		WithSetHeader(headerAPIKey, key).
		WithSetHeader(headerAPISecret, secret)
}

// StoreProperty attempts to store a property using the RPM API
// when the response code is >=400 you will get an ErrorResponse as the error
func (c Client) StoreProperty(ctx context.Context, r StorePropertyReq) (*StorePropertyRes, error) {
	var out StorePropertyRes
	if err := c.do(ctx, http.MethodPost, &r, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (c Client) do(ctx context.Context, method string, r httputil.Request, out any) error {
	var er ErrorResponse
	_, err := c.client.DoReq(ctx, method, r, &out, &er)
	if err != nil {
		return err
	}
	if err := er.ErrorOrNil(); err != nil {
		return err
	}
	return nil
}
