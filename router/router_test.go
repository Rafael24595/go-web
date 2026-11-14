package router

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rafael24595/go-assert/assert"
	"github.com/Rafael24595/go-assert/require"
	"github.com/Rafael24595/go-web/router/docs"
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

	body, err := io.ReadAll(res.Body)
	assert.Nil(t, err)
	assert.Nil(t, res.Body.Close())

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

type recordingViewer struct {
	docs.IDocViewer
	routes []string
}

func (v *recordingViewer) RegisterRoute(route docs.DocOperation) docs.IDocViewer {
	v.routes = append(v.routes, route.Method+" "+route.Path)
	return v
}

func recordGroup(calls *[]string, name string) RequestHandler {
	return func(w http.ResponseWriter, req *http.Request, ctx *Context) result.Result {
		*calls = append(*calls, name)
		return result.Continue()
	}
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

	body, err := io.ReadAll(res.Body)
	assert.Nil(t, err)

	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
	assert.Equal(t, "nosniff", res.Header.Get("X-Content-Type-Options"))

	assert.Inside(t, `"error": "missing"`, string(body))

	assert.Nil(t, res.Body.Close())
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

	assert.Equal(t, http.StatusTeapot, res.StatusCode)

	assert.NotNil(t, handled)
	assert.Equal(t, http.StatusConflict, handled.Status())

	assert.Nil(t, res.Body.Close())
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

	body, err := io.ReadAll(res.Body)
	assert.Nil(t, err)

	assert.False(t, reached)

	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
	assert.Inside(t, "no token", string(body))

	assert.Nil(t, res.Body.Close())
}

func TestRouterUse(t *testing.T) {
	origin := "https://app.example.com"

	ok := func(w http.ResponseWriter, req *http.Request, ctx *Context) result.Result {
		return result.TextOk("ok")
	}

	wrap := func(calls *[]string, name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				*calls = append(*calls, name)
				next.ServeHTTP(w, req)
			})
		}
	}

	forbid := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "forbidden", http.StatusForbidden)
		})
	}

	t.Run("the first middleware added is the outermost", func(t *testing.T) {
		var calls []string

		router := NewRouterWithMux(nil).
			Use(
				wrap(&calls, "first"),
				nil,
				wrap(&calls, "second"),
			).
			Use(wrap(&calls, "third")).
			Route(http.MethodGet, ok, "/x")

		recorder := httptest.NewRecorder()
		router.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/x", nil))

		want := []string{"first", "second", "third"}

		require.Size(t, len(want), calls)

		for i := range want {
			assert.Equal(t, want[i], calls[i])
		}

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("a middleware can stop the request", func(t *testing.T) {
		router := NewRouterWithMux(nil).
			Use(forbid).
			Route(http.MethodGet, ok, "/x")

		recorder := httptest.NewRecorder()
		router.Handler().
			ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/x", nil))

		assert.Equal(t, http.StatusForbidden, recorder.Code)
	})

	t.Run("a preflight is answered before the middlewares", func(t *testing.T) {
		cors := NewCors().
			AllowedOrigins(origin).
			AllowedMethods("GET")

		router := NewRouterWithMux(nil).
			Cors(cors).
			Use(forbid).
			Route(http.MethodGet, ok, "/x")

		req := httptest.NewRequest(http.MethodOptions, "/x", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)

		recorder := httptest.NewRecorder()
		router.Handler().ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusNoContent, recorder.Code)
		assertHeader(t, recorder, "Access-Control-Allow-Origin", origin)
	})

	t.Run("a rejection carries the CORS headers", func(t *testing.T) {
		cors := NewCors().
			AllowedOrigins(origin).
			AllowedMethods("GET")

		router := NewRouterWithMux(nil).
			Cors(cors).
			Use(forbid).
			Route(http.MethodGet, ok, "/x")

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", origin)

		recorder := httptest.NewRecorder()
		router.Handler().ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusForbidden, recorder.Code)
		assertHeader(t, recorder, "Access-Control-Allow-Origin", origin)
	})
}

func TestRouteWithOptions_OnlyDocumentedRoutesReachTheViewer(t *testing.T) {
	viewer := &recordingViewer{IDocViewer: docs.VoidViewer()}
	id := docs.DocOrderParameters{docs.Parameter("id", "an id")}

	NewRouterWithMux(nil).
		DocViewer(viewer).
		Route(http.MethodGet, text("ok"), "/plain").
		RouteWithOptions(http.MethodGet, text("ok"), "/options", WithPanic(nil)).
		RouteDocument(http.MethodGet, text("ok"), "/documented", docs.DocRoute{}).
		RouteWithOptions(http.MethodGet, text("ok"), "/documented/{%s}", WithDocs(docs.DocRoute{Parameters: id}))

	assert.DeepEqual(t, []string{"GET /documented", "GET /documented/{id}"}, viewer.routes)
}

func TestRouteWithOptions_Params(t *testing.T) {
	router := NewRouterWithMux(nil).
		RouteWithOptions(http.MethodGet, text("item"), "/%s/{%s}", WithParams("items", "id"))

	res, body := get(t, serve(t, router).URL+"/items/42")

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "item", body)
}

func TestRouteWithOptions_HandlersOfTheRoute(t *testing.T) {
	routeError := func(w http.ResponseWriter, req *http.Request, ctx *Context, res result.Result) {
		http.Error(w, "route error", http.StatusTeapot)
	}

	routePanic := func(w http.ResponseWriter, req *http.Request, rec any) {
		http.Error(w, "route panic", http.StatusServiceUnavailable)
	}

	globalError := func(w http.ResponseWriter, req *http.Request, ctx *Context, res result.Result) {
		http.Error(w, "global error", http.StatusInternalServerError)
	}

	router := NewRouterWithMux(nil).
		ErrorHandler(globalError).
		RouteWithOptions(http.MethodGet, bad, "/own-error", WithError(routeError)).
		RouteWithOptions(http.MethodGet, boom, "/own-panic", WithPanic(routePanic)).
		Route(http.MethodGet, bad, "/global-error")

	server := serve(t, router)

	res, _ := get(t, server.URL+"/own-error")
	assert.Equal(t, http.StatusTeapot, res.StatusCode)

	res, _ = get(t, server.URL+"/own-panic")
	assert.Equal(t, http.StatusServiceUnavailable, res.StatusCode)

	res, _ = get(t, server.URL+"/global-error")
	assert.Equal(t, http.StatusInternalServerError, res.StatusCode, "other routes keep the global handler")
}

func TestGroupContextualizer_BasePath(t *testing.T) {
	var calls []string

	router := NewRouterWithMux(nil).
		BasePath("/v1").
		GroupContextualizer(
			recordGroup(&calls, "api"), "/api",
		).
		Route(http.MethodGet, text("ok"), "/api/x")

	server := serve(t, router)
	res, _ := get(t, server.URL+"/v1/api/x")

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.DeepEqual(t, []string{"api"}, calls)
}

func TestGroupContextualizer_WholeSegments(t *testing.T) {
	var calls []string

	router := NewRouterWithMux(nil).
		GroupContextualizer(
			recordGroup(&calls, "api"), "/api",
		).
		Route(http.MethodGet, text("ok"), "/apiary")

	server := serve(t, router)
	get(t, server.URL+"/apiary")

	assert.Size(t, 0, calls)
}

func TestGroupContextualizer_OuterGroupsFirst(t *testing.T) {
	for range 20 {
		var calls []string

		router := NewRouterWithMux(nil).
			GroupContextualizer(
				recordGroup(&calls, "admin"), "/api/admin",
			).
			GroupContextualizer(
				recordGroup(&calls, "api"), "/api",
			).
			Route(http.MethodGet, text("ok"), "/api/admin/users")

		server := serve(t, router)
		get(t, server.URL+"/api/admin/users")

		assert.DeepEqual(t, []string{"api", "admin"}, calls)
	}
}
