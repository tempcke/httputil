package httputil_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tempcke/httputil"
	"github.com/tempcke/httputil/example"
)

var (
	ctx       = context.Background()
	errLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
)

type (
	ReqModel struct {
		Name string `json:"name"`
	}
	ResModel struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	ErrResModel struct {
		Err struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
)

func TestClient_Do(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		var (
			called    = false
			method    = http.MethodPost
			uri       = "/foo/" + uuid.NewString()
			in        = ReqModel{Name: "Robert"}
			reqHeader http.Header
		)
		server := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			called = true
			assert.Equal(t, method, r.Method)
			assert.Equal(t, uri, r.URL.Path)
			w.WriteHeader(http.StatusOK)
		})
		client := httputil.NewClient().
			WithLogger(errLogger).WithHost(server.URL)

		fullPath := httputil.NewPath(uri).WithBaseURL(server.URL).String()

		req, err := client.Request(ctx, method, fullPath, reqHeader, in)
		require.NoError(t, err)
		res, err := client.Do(req)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.True(t, called)
	})
	t.Run("headers", func(t *testing.T) {
		// two ways to add headers to the client: WithHeader and AddHeader
		// can also set headers per request
		var (
			called         = false
			method         = http.MethodPost
			uri            = "/foo/" + uuid.NewString()
			fooKey, fooVal = "X-Foo", "foo"
			barKey, barVal = "X-Bar", "bar"
			bazKey, bazVal = "X-Baz", "baz"
			reqHeader      = http.Header{
				fooKey: []string{fooVal},
			}
			in = ReqModel{Name: "Robert"}
		)
		server := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			called = true
			assert.Equal(t, fooVal, r.Header.Get(fooKey))
			assert.Equal(t, barVal, r.Header.Get(barKey))
			assert.Equal(t, bazVal, r.Header.Get(bazKey))
			w.WriteHeader(http.StatusOK)
		})
		client := httputil.NewClient().WithHost(server.URL).
			WithHeader(http.Header{bazKey: []string{bazVal}})
		client.AddHeader(barKey, barVal)

		fullPath := httputil.NewPath(uri).WithBaseURL(server.URL).String()
		req, err := client.Request(ctx, method, fullPath, reqHeader, in)
		require.NoError(t, err)
		res, err := client.Do(req)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.True(t, called)
	})
}
func TestClient_DoAndDecode(t *testing.T) {
	var (
		called    = false
		method    = http.MethodPost
		uri       = "/foo/" + uuid.NewString()
		in        = ReqModel{Name: "Robert"}
		reqHeader http.Header
		id        = uuid.NewString()
	)
	t.Run("ok response", func(t *testing.T) {
		server := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			called = true
			assert.Equal(t, method, r.Method)
			assert.Equal(t, uri, r.URL.Path)
			w.WriteHeader(http.StatusOK)
			writeJSON(w, ResModel{
				ID:   id,
				Name: in.Name,
			})
		})
		client := httputil.NewClient().
			WithLogger(errLogger).WithHost(server.URL)

		fullPath := httputil.NewPath(uri).WithBaseURL(server.URL).String()

		req, err := client.Request(ctx, method, fullPath, reqHeader, in)
		require.NoError(t, err)
		var (
			out    ResModel
			errRes ErrResModel
		)
		res, err := client.DoAndDecode(req, &out, &errRes)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.True(t, called)
		assert.Equal(t, id, out.ID)
		assert.Equal(t, in.Name, out.Name)
	})
	t.Run("error response", func(t *testing.T) {
		var (
			errCode = 42
			errMsg  = uuid.NewString()
		)
		server := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			called = true
			assert.Equal(t, method, r.Method)
			assert.Equal(t, uri, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]any{
				"error": map[string]any{
					"code":    errCode,
					"message": errMsg,
				},
			})
		})
		client := httputil.NewClient().
			WithLogger(errLogger).WithHost(server.URL)

		fullPath := httputil.NewPath(uri).WithBaseURL(server.URL).String()

		req, err := client.Request(ctx, method, fullPath, reqHeader, in)
		require.NoError(t, err)
		var (
			out    ResModel
			errRes ErrResModel
		)
		res, err := client.DoAndDecode(req, &out, &errRes)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.True(t, called)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "", out.ID)
		assert.Equal(t, "", out.Name)
		assert.Equal(t, errCode, errRes.Err.Code)
		assert.Equal(t, errMsg, errRes.Err.Message)
	})
}
func TestClient_DoReq(t *testing.T) {
	var (
		method   = http.MethodPost
		property = example.Property{
			ID:     uuid.NewString(),
			Street: "1901 Main st",
			City:   "Dallas",
			State:  "TX",
			Zip:    "75201",
		}
		req    = example.NewStorePropertyReq(property)
		fooKey = "x-foo"
		fooVal = "foo"
	)
	req.ReqID = reqID
	req.AddHeader(fooKey, fooVal)
	t.Run("ok response", func(t *testing.T) {
		var called = false
		client := clientWithFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			called = true

			// method and path
			assert.Equal(t, method, r.Method)
			assert.Equal(t, req.Path().String(), r.URL.Path)

			// body
			var in example.StorePropertyReq
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&in))
			assert.Equal(t, req.Property.Street, in.Property.Street)
			assert.Equal(t, req.Property.City, in.Property.City)
			assert.Equal(t, req.Property.State, in.Property.State)
			assert.Equal(t, req.Property.Zip, in.Property.Zip)

			// headers
			assert.Equal(t, reqID, r.Header.Get(httputil.HeaderReqID))
			assert.Equal(t, fooVal, r.Header.Get(fooKey))

			w.WriteHeader(http.StatusOK)
			writeJSON(w, example.StorePropertyRes{
				Property: property,
			})
		})

		var (
			out    example.StorePropertyRes
			errRes ErrResModel
		)
		res, err := client.DoReq(ctx, method, &req, &out, &errRes)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.True(t, called)
		assert.Equal(t, property, out.Property)
	})
	t.Run("error response", func(t *testing.T) {
		var (
			apiErr = example.APIError{
				Code:    42,
				Message: uuid.NewString(),
			}
			client = clientWithFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				writeJSON(w, example.ErrorResponse{APIError: apiErr})
			})
			out    example.StorePropertyRes
			errRes example.ErrorResponse
		)
		res, err := client.DoReq(ctx, method, &req, &out, &errRes)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, apiErr, errRes.APIError)
	})
	t.Run("validation error", func(t *testing.T) {
		var (
			called = false
			client = clientWithFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
				called = true
			})
			out    example.StorePropertyRes
			errRes example.ErrorResponse
			req    = req
		)
		req.Property.Street = ""
		res, err := client.DoReq(ctx, method, &req, &out, &errRes)
		assert.False(t, called)
		assert.ErrorIs(t, err, httputil.ErrInvalidRequest)
		assert.ErrorIs(t, err, example.ErrStreetRequired)
		assert.Nil(t, res)
	})
}

func TestClient_RateLimit(t *testing.T) {
	var (
		reqPerSec   = 1.0
		callsToMake = int(reqPerSec + 1)
		callCtr     = atomic.Int32{}
		limiter     = httputil.NewRateLimiter(reqPerSec)
	)
	client := clientWithFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCtr.Add(1)
		w.WriteHeader(http.StatusOK)
	}).WithRateLimiter(limiter)

	req, err := client.Request(ctx, http.MethodGet, "/", nil, nil)
	require.NoError(t, err)

	start := time.Now()
	for i := 0; i < callsToMake; i++ {
		_, err = client.Do(req)
		require.NoError(t, err)
	}
	if d := time.Since(start); d < time.Second {
		t.Errorf("expected at least %s, got %s", time.Second.String(), d.String())
	}
	assert.Equal(t, callCtr.Load(), int32(callsToMake))
}
func TestClient_RetryOn429(t *testing.T) {
	// when want to be able to configure the client to retry on 429
	var (
		reqPerSec     = 100.0
		callCtr       = atomic.Int32{}
		limiter       = httputil.NewRateLimiter(reqPerSec)
		retries       = 2
		expectCallCtr = int32(retries + 1)
	)

	client := clientWithFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCtr.Add(1)
		if callCtr.Load() <= int32(retries) {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}).WithRateLimiter(limiter).With429Retry(retries)

	req, err := client.Request(ctx, http.MethodGet, "/", nil, nil)
	require.NoError(t, err)

	res, err := client.Do(req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, expectCallCtr, callCtr.Load())
}
func TestClient_SlowDownOn429(t *testing.T) {
	var (
		reqPerSec     = 100.0
		callCtr       = atomic.Int32{}
		limiter       = httputil.NewRateLimiter(reqPerSec)
		retries       = 0
		expectCallCtr = int32(retries + 1)
	)

	client := clientWithFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCtr.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}).WithRateLimiter(limiter).With429Retry(retries)

	req, err := client.Request(ctx, http.MethodGet, "/", nil, nil)
	require.NoError(t, err)

	assert.Equal(t, reqPerSec, limiter.Limit())
	res, err := client.Do(req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, http.StatusTooManyRequests, res.StatusCode)
	assert.Equal(t, expectCallCtr, callCtr.Load())

	// expect the reqPerSec to have decreased because of the 429
	assert.Greater(t, reqPerSec, limiter.Limit())
}

func clientWithFakeServer(t testing.TB, h func(w http.ResponseWriter, r *http.Request)) httputil.Client {
	server := fakeServer(t, h)
	t.Cleanup(server.Close)
	return httputil.NewClient().WithHost(server.URL).WithLogger(errLogger)
}
func fakeServer(t testing.TB, h func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()
		h(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}
func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	out, _ := json.Marshal(data)
	_, _ = w.Write(out)
}
