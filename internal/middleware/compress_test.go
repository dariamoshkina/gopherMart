package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var helloHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	w.Write(append([]byte("hello "), body...))
})

func TestCompress_CompressesResponse(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	Compress(helloHandler).ServeHTTP(rec, r)

	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))

	gr, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	defer gr.Close()
	body, err := io.ReadAll(gr)
	require.NoError(t, err)
	assert.Equal(t, "hello ", string(body))
}

func TestCompress_NoCompressionWithoutHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	Compress(helloHandler).ServeHTTP(rec, r)

	assert.Empty(t, rec.Header().Get("Content-Encoding"))
	assert.Equal(t, "hello ", rec.Body.String())
}

func TestCompress_DecompressesRequestBody(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write([]byte("world"))
	gz.Close()

	r := httptest.NewRequest(http.MethodPost, "/", &buf)
	r.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	Compress(helloHandler).ServeHTTP(rec, r)

	assert.Equal(t, "hello world", rec.Body.String())
}

func TestCompress_BadGzipBody_Returns400(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not-gzip"))
	r.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	Compress(helloHandler).ServeHTTP(rec, r)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
