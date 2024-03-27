package httputil_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tempcke/httputil"
)

func TestDecodeResponse(t *testing.T) {
	t.Run("nil out", func(t *testing.T) {
		resBodyBytes := `{"foo": "bar"}`
		res := &http.Response{
			Body: io.NopCloser(bytes.NewBufferString(resBodyBytes)),
		}
		bodyBytes, err := httputil.DecodeResponse(res, nil)
		require.NoError(t, err)
		require.Equal(t, resBodyBytes, string(bodyBytes))
	})
}
