package example

import (
	"context"
	"net/http"

	"github.com/tempcke/httputil"
)

type RPM struct {
	client httputil.Client
}

// StoreProperty attempts to store a property using the RPM API
// when the response code is >=400 you will get an ErrorResponse as the error
func (c RPM) StoreProperty(ctx context.Context, r StorePropertyReq) (*StorePropertyRes, error) {
	var out StorePropertyRes
	if err := c.do(ctx, http.MethodPost, &r, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (c RPM) do(ctx context.Context, method string, r httputil.Request, out any) error {
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
