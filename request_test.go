package httputil_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tempcke/httputil"
)

var (
	reqID = uuid.NewString()
)

func TestRequest_NewRequest(t *testing.T) {
	// so the idea is that a normal request model can embed ReqHeaders
	// which will allow the user to add custom headers to any request
	// you could also implement your own Header() method in order to return
	// headers from the request struct fields
	var (
		fooKey, fooVal = "X-Foo", "foo"
		barKey, barVal = "X-Bar", "bar"
		bazKey, bazVal = "X-Baz", "baz"
		p              = Person{
			ID:        "uncle-bob",
			FirstName: "Robert",
			LastName:  "Martin",
		}
		req = StorePersonReq{Person: p}
	)
	req.AddHeader(fooKey, fooVal)
	req.AddHeader(fooKey, bazVal) // should append
	req.SetHeader(barKey, barVal)
	req.SetHeader(barKey, bazVal) // should replace
	req.SetHeader(bazKey, bazVal)
	req.ReqID = reqID

	assert.Equal(t, fooVal, req.Header().Get(fooKey))
	assert.Equal(t, []string{fooVal, bazVal}, req.Header()[fooKey])
	assert.Equal(t, bazVal, req.Header().Get(barKey))
	assert.Equal(t, bazVal, req.Header().Get(bazKey))
	assert.Equal(t, reqID, req.Header().Get(httputil.HeaderReqID))
}

type (
	Person struct {
		ID        string `json:"id"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	}
	StorePersonReq struct {
		httputil.ReqHeaders
		Person Person `json:"person"`
	}
)

var _ httputil.Request = (*StorePersonReq)(nil)

func (r StorePersonReq) Path() httputil.Path {
	return httputil.NewPath("/person/:personID").
		WithParam(":personID", "P")
}
func (r StorePersonReq) Validate() error {
	if r.Person.ID == "" {
		return errors.New("missing ID")
	}
	if r.Person.FirstName == "" {
		return errors.New("missing first name")
	}
	if r.Person.LastName == "" {
		return errors.New("missing last name")
	}
	return nil
}
