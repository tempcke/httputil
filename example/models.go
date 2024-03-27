package example

import (
	"errors"
	"fmt"

	"github.com/tempcke/httputil"
)

type (
	Property struct {
		ID     string `json:"id"`
		Street string `json:"street"`
		City   string `json:"city"`
		State  string `json:"state"`
		Zip    string `json:"zip"`
	}
	StorePropertyReq struct {
		httputil.ReqHeaders `json:"-"`
		Property            struct {
			ID     string `json:"-"` // path arg
			Street string `json:"street"`
			City   string `json:"city"`
			State  string `json:"state"`
			Zip    string `json:"zip"`
		} `json:"property"`
	}
	StorePropertyRes struct {
		Property `json:"property"`
	}
	ErrorResponse struct {
		APIError APIError `json:"error"`
	}
	APIError struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Type    string `json:"type"`
	}
)

var (
	ErrStreetRequired = errors.New("missing street")
)

func NewStorePropertyReq(p Property) StorePropertyReq {
	r := StorePropertyReq{}
	r.Property.ID = p.ID
	r.Property.Street = p.Street
	r.Property.City = p.City
	r.Property.State = p.State
	r.Property.Zip = p.Zip
	return r
}
func (req StorePropertyReq) Validate() error {
	switch {
	case req.Property.Street == "":
		return ErrStreetRequired
	case req.Property.City == "":
		return errors.New("missing city")
	case req.Property.State == "":
		return errors.New("missing state")
	case req.Property.Zip == "":
		return errors.New("missing zip")
	}
	return nil
}
func (req StorePropertyReq) Path() httputil.Path {
	return httputil.NewPath("/property/:propertyID").
		WithParam(":propertyID", req.Property.ID)
}
func (e ErrorResponse) Error() string {
	if e.APIError.Message == "" {
		return ""
	}
	return fmt.Sprintf("RPM error: %s (%d) %s", e.APIError.Type, e.APIError.Code, e.APIError.Message)
}
func (e ErrorResponse) IsZero() bool {
	return e.APIError.Code == 0 && e.APIError.Message == "" && e.APIError.Type == ""
}

func (e ErrorResponse) ErrorOrNil() error {
	if e.IsZero() {
		return nil
	}
	return e
}
