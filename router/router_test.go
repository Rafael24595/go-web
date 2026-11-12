package router

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rafael24595/go-assert/assert"
	"github.com/Rafael24595/go-web/router/result"
)

func serve(t *testing.T, router *Router) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(router.Handler())
	t.Cleanup(server.Close)

	return server
}

func get(t *testing.T, url string) (*http.Response, string) {
	t.Helper()

	res, err := http.Get(url)
	assert.Nil(t, err)

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	assert.Nil(t, err)

	return res, string(body)
}

func text(message string) RequestHandler {
	return func(w http.ResponseWriter, req *http.Request, ctx *Context) result.Result {
		return result.TextOk(message)
	}
}

func boom(w http.ResponseWriter, req *http.Request, ctx *Context) result.Result {
	panic("boom")
}

func bad(w http.ResponseWriter, req *http.Request, ctx *Context) result.Result {
	return result.TextErr(http.StatusBadRequest, "bad input")
}

func TestNormalizeAddr(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want string
	}{
		{
			name: "empty address",
			addr: "",
			want: "",
		},
		{
			name: "port only",
			addr: "8080",
			want: ":8080",
		},
		{
			name: "port with leading colon",
			addr: ":8080",
			want: ":8080",
		},
		{
			name: "localhost with port",
			addr: "localhost:8080",
			want: "localhost:8080",
		},
		{
			name: "IPv4 with port",
			addr: "127.0.0.1:8080",
			want: "127.0.0.1:8080",
		},
		{
			name: "IPv6 with port",
			addr: "[::1]:8080",
			want: "[::1]:8080",
		},
		{
			name: "hostname without port",
			addr: "localhost",
			want: "localhost",
		},
		{
			name: "IPv6 without port",
			addr: "::1",
			want: "::1",
		},
		{
			name: "non numeric address",
			addr: "http",
			want: "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAddr(tt.addr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRouter_OwnMuxPerRouter(t *testing.T) {
	first := NewRouterWithMux(http.NewServeMux()).
		Route(http.MethodGet, text("first"), "/x")

	second := NewRouterWithMux(http.NewServeMux()).
		Route(http.MethodGet, text("second"), "/x")

	server := serve(t, first)
	_, body := get(t, server.URL+"/x")
	assert.Equal(t, "first", body)

	server = serve(t, second)
	_, body = get(t, server.URL+"/x")
	assert.Equal(t, "second", body)

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	_, pattern := http.DefaultServeMux.Handler(req)

	assert.Equal(t, "", pattern)
}

func TestRouter_NilMuxCreatesOne(t *testing.T) {
	router := NewRouterWithMux(nil).
		Route(http.MethodGet, text("ok"), "/x")

	server := serve(t, router)
	res, body := get(t, server.URL+"/x")

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "ok", body)
}

func TestRouter_HeadersReachTheClient(t *testing.T) {
	handler := func(w http.ResponseWriter, req *http.Request, ctx *Context) result.Result {
		w.Header().Set("ETag", `"v1"`)
		return result.JsonOks(http.StatusCreated, map[string]int{"a": 1})
	}

	router := NewRouterWithMux(http.NewServeMux()).
		Route(http.MethodGet, handler, "/x")

	server := serve(t, router)
	res, _ := get(t, server.URL+"/x")

	assert.Equal(t, http.StatusCreated, res.StatusCode)
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
	assert.Equal(t, `"v1"`, res.Header.Get("ETag"))
}

func TestRouter_PanicHandler(t *testing.T) {
	var recovered any

	panicHandler := func(w http.ResponseWriter, req *http.Request, rec any) {
		recovered = rec
		http.Error(w, "handled", http.StatusServiceUnavailable)
	}

	router := NewRouterWithMux(http.NewServeMux()).
		PanicHandler(panicHandler).
		Route(http.MethodGet, boom, "/x")

	server := serve(t, router)
	res, _ := get(t, server.URL+"/x")

	assert.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
	assert.Equal(t, "boom", recovered)
}

func TestRouter_PanicWithoutHandler(t *testing.T) {
	router := NewRouterWithMux(http.NewServeMux()).
		Route(http.MethodGet, boom, "/x")

	server := serve(t, router)
	res, _ := get(t, server.URL+"/x")

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
}

func TestRouter_PatternWithoutHandlerIsNotFound(t *testing.T) {
	router := NewRouterWithMux(http.NewServeMux())

	req := httptest.NewRequest(http.MethodGet, "/nowhere", nil)

	recorder := httptest.NewRecorder()
	router.handler(recorder, req)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestRouter_ErrorHandler(t *testing.T) {
	var handled result.Result

	errorHandler := func(w http.ResponseWriter, req *http.Request, ctx *Context, res result.Result) {
		handled = res
		http.Error(w, "custom", http.StatusTeapot)
	}

	router := NewRouterWithMux(http.NewServeMux()).
		ErrorHandler(errorHandler).
		Route(http.MethodGet, bad, "/x")

	server := serve(t, router)
	res, body := get(t, server.URL+"/x")

	assert.Equal(t, http.StatusTeapot, res.StatusCode)
	assert.Equal(t, "custom\n", body)
	assert.Equal(t, http.StatusBadRequest, handled.Status())
}

func TestRouter_ErrorWithoutHandlerIsPlainText(t *testing.T) {
	router := NewRouterWithMux(http.NewServeMux()).
		Route(http.MethodGet, bad, "/x")

	res, body := get(t, serve(t, router).URL+"/x")

	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	assert.Equal(t, "bad input", body)
	assert.Equal(t, "text/plain; charset=utf-8", res.Header.Get("Content-Type"))
}

func TestRouter_JsonErrWithoutErrorHandler(t *testing.T) {
	handler := func(w http.ResponseWriter, req *http.Request, ctx *Context) result.Result {
		return result.JsonErr(http.StatusNotFound, map[string]string{"error": "missing"})
	}

	router := NewRouterWithMux(http.NewServeMux()).
		Route(http.MethodGet, handler, "/x")

	server := httptest.NewServer(router.Handler())
	defer server.Close()

	res, err := http.Get(server.URL + "/x")
	assert.Nil(t, err)

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	assert.Nil(t, err)

	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
	assert.Equal(t, "nosniff", res.Header.Get("X-Content-Type-Options"))

	assert.Inside(t, `"error": "missing"`, string(body))
}

func TestRouter_JsonErrReachesTheErrorHandler(t *testing.T) {
	var handled *result.Result

	errorHandler := func(w http.ResponseWriter, req *http.Request, ctx *Context, res result.Result) {
		handled = &res
		w.WriteHeader(http.StatusTeapot)
	}

	handler := func(w http.ResponseWriter, req *http.Request, ctx *Context) result.Result {
		return result.JsonErr(http.StatusConflict, map[string]string{"error": "stale"})
	}

	router := NewRouterWithMux(http.NewServeMux()).
		ErrorHandler(errorHandler).
		Route(http.MethodGet, handler, "/x")

	server := httptest.NewServer(router.Handler())
	defer server.Close()

	res, err := http.Get(server.URL + "/x")
	assert.Nil(t, err)

	defer res.Body.Close()

	assert.Equal(t, http.StatusTeapot, res.StatusCode)

	assert.NotNil(t, handled)
	assert.Equal(t, http.StatusConflict, handled.Status())
}

func TestRouter_JsonErrInGroupContextualizerStopsTheRequest(t *testing.T) {
	reached := false

	auth := func(w http.ResponseWriter, req *http.Request, ctx *Context) result.Result {
		return result.JsonErr(http.StatusUnauthorized, map[string]string{"error": "no token"})
	}

	handler := func(w http.ResponseWriter, req *http.Request, ctx *Context) result.Result {
		reached = true
		return result.JsonOk(map[string]string{"secret": "data"})
	}

	router := NewRouterWithMux(http.NewServeMux()).
		GroupContextualizer(auth, "/private").
		Route(http.MethodGet, handler, "/private/x")

	server := httptest.NewServer(router.Handler())
	defer server.Close()

	res, err := http.Get(server.URL + "/private/x")
	assert.Nil(t, err)

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	assert.Nil(t, err)

	assert.False(t, reached)

	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
	assert.Inside(t, "no token", string(body))
}
