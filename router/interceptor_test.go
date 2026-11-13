package router

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rafael24595/go-assert/assert"
	"github.com/Rafael24595/go-assert/require"
	"github.com/Rafael24595/go-web/router/result"
)

func run(interceptor interceptor, req *http.Request) (bool, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	stop := interceptor(recorder, req)
	return stop, recorder
}

func assertHeader(t *testing.T, recorder *httptest.ResponseRecorder, header, want string) {
	t.Helper()
	assert.Equal(t, want, recorder.Header().Get(header))
}

func TestHttpsRedirectInterceptor(t *testing.T) {
	tests := []struct {
		name     string
		addrTLS  string
		url      string
		host     string
		location string
	}{
		{
			name:     "port of the TLS server",
			addrTLS:  ":8443",
			url:      "/x?a=1",
			host:     "example.com:8080",
			location: "https://example.com:8443/x?a=1",
		},
		{
			name:     "host without port",
			addrTLS:  ":8443",
			url:      "/x",
			host:     "example.com",
			location: "https://example.com:8443/x",
		},
		{
			name:     "default HTTPS port is omitted",
			addrTLS:  ":443",
			url:      "/x",
			host:     "example.com:8080",
			location: "https://example.com/x",
		},
		{
			name:     "listen address is not the public host",
			addrTLS:  "0.0.0.0:8443",
			url:      "/x",
			host:     "example.com:8080",
			location: "https://example.com:8443/x",
		},
		{
			name:     "IPv6 host",
			addrTLS:  ":8443",
			url:      "/x",
			host:     "[::1]:8080",
			location: "https://[::1]:8443/x",
		},
		{
			name:     "absolute-form request",
			addrTLS:  ":8443",
			url:      "http://example.com:8080/x?a=1",
			host:     "",
			location: "https://example.com:8443/x?a=1",
		},
		{
			name:     "address without port",
			addrTLS:  "example.com",
			url:      "/x",
			host:     "example.com:8080",
			location: "https://example.com/x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
			if tt.host != "" {
				req.Host = tt.host
			}

			stop, recorder := run(
				httpsRedirectInterceptor(tt.addrTLS), req,
			)

			assert.True(t, stop)
			assert.Equal(t, http.StatusPermanentRedirect, recorder.Code)
			assert.Equal(t, tt.location, recorder.Header().Get("Location"))
		})
	}
}

func TestHttpsRedirectInterceptor_LetsTLSRequestsThrough(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.TLS = &tls.ConnectionState{}

	stop, recorder := run(
		httpsRedirectInterceptor(":8443"), req,
	)

	assert.False(t, stop)
	assert.Equal(t, "", recorder.Header().Get("Location"))
}

func TestCorsInterceptor(t *testing.T) {
	listed := NewCors().
		AllowedOrigins("https://a.example.com", "https://b.example.com").
		AllowedMethods("GET", "PUT").
		AllowedHeaders("Content-Type", "Authorization").
		AllowCredentials()

	t.Run("a listed origin is echoed alone", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", "https://b.example.com")

		stop, recorder := run(
			corsInterceptor(*listed), req,
		)

		assert.False(t, stop)

		assertHeader(t, recorder, "Access-Control-Allow-Origin", "https://b.example.com")
		assertHeader(t, recorder, "Access-Control-Allow-Credentials", "true")
		assertHeader(t, recorder, "Access-Control-Allow-Methods", "")

		assertHeader(t, recorder, "Vary", "Origin")
	})

	t.Run("an unlisted origin gets no CORS headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", "https://evil.example.com")

		stop, recorder := run(
			corsInterceptor(*listed), req,
		)

		assert.False(t, stop)

		assertHeader(t, recorder, "Access-Control-Allow-Origin", "")
		assertHeader(t, recorder, "Access-Control-Allow-Credentials", "")

		assertHeader(t, recorder, "Vary", "Origin")
	})

	t.Run("a request without Origin is not a CORS request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)

		stop, recorder := run(
			corsInterceptor(*listed), req,
		)

		assert.False(t, stop)
		assertHeader(t, recorder, "Access-Control-Allow-Origin", "")
	})

	t.Run("wildcard without credentials answers *", func(t *testing.T) {
		cors := NewCors().
			AllowedOrigins("*").
			AllowedMethods("GET")

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", "https://any.example.com")

		_, recorder := run(
			corsInterceptor(*cors), req,
		)

		assertHeader(t, recorder, "Access-Control-Allow-Origin", "*")
		assertHeader(t, recorder, "Access-Control-Allow-Credentials", "")
	})

	t.Run("wildcard with credentials echoes the origin", func(t *testing.T) {
		cors := NewCors().
			AllowedOrigins("*").
			AllowCredentials()

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", "https://any.example.com")

		_, recorder := run(
			corsInterceptor(*cors), req,
		)

		assertHeader(t, recorder, "Access-Control-Allow-Origin", "https://any.example.com")
		assertHeader(t, recorder, "Access-Control-Allow-Credentials", "true")
	})

	t.Run("PermissiveCors allows any origin without credentials", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/x", nil)
		req.Header.Set("Origin", "https://any.example.com")
		req.Header.Set("Access-Control-Request-Method", "PUT")

		stop, recorder := run(
			corsInterceptor(*PermissiveCors()), req,
		)

		assert.True(t, stop)
		assertHeader(t, recorder, "Access-Control-Allow-Origin", "*")
		assertHeader(t, recorder, "Access-Control-Allow-Credentials", "")
		assertHeader(t, recorder, "Access-Control-Allow-Headers", "Content-Type, Authorization")
	})

	t.Run("credentials are not announced unless enabled", func(t *testing.T) {
		cors := NewCors().
			AllowedOrigins("https://a.example.com")

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", "https://a.example.com")

		_, recorder := run(
			corsInterceptor(*cors), req,
		)

		assertHeader(t, recorder, "Access-Control-Allow-Credentials", "")
	})

	t.Run("preflight is answered and stops the chain", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/x", nil)
		req.Header.Set("Origin", "https://a.example.com")
		req.Header.Set("Access-Control-Request-Method", "PUT")

		stop, recorder := run(
			corsInterceptor(*listed), req,
		)

		assert.True(t, stop)
		assert.Equal(t, http.StatusNoContent, recorder.Code)

		assertHeader(t, recorder, "Access-Control-Allow-Origin", "https://a.example.com")
		assertHeader(t, recorder, "Access-Control-Allow-Methods", "GET, PUT")
		assertHeader(t, recorder, "Access-Control-Allow-Headers", "Content-Type, Authorization")
	})

	t.Run("a preflight from an unlisted origin reaches the routes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/x", nil)
		req.Header.Set("Origin", "https://evil.example.com")
		req.Header.Set("Access-Control-Request-Method", "PUT")

		stop, recorder := run(
			corsInterceptor(*listed), req,
		)

		assert.False(t, stop)
		assertHeader(t, recorder, "Access-Control-Allow-Methods", "")
	})

	t.Run("OPTIONS without a preflight header reaches the routes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/x", nil)
		req.Header.Set("Origin", "https://any.example.com")

		stop, _ := run(
			corsInterceptor(*listed), req,
		)

		assert.False(t, stop)
	})
}

func TestApplyInterceptor(t *testing.T) {
	record := func(calls *[]string, name string, stop bool) interceptor {
		return func(w http.ResponseWriter, req *http.Request) bool {
			*calls = append(*calls, name)
			return stop
		}
	}

	next := func(calls *[]string) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			*calls = append(*calls, "handler")
		})
	}

	t.Run("runs in order and reaches the handler", func(t *testing.T) {
		var calls []string

		handler := applyInterceptor(
			next(&calls),
			record(&calls, "first", false),
			record(&calls, "second", false),
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)

		want := []string{"first", "second", "handler"}

		require.Size(t, len(want), calls)

		for i := range want {
			assert.Equal(t, want[i], calls[i])
		}
	})

	t.Run("a interceptor that stops skips the rest", func(t *testing.T) {
		var calls []string

		handler := applyInterceptor(
			next(&calls),
			record(&calls, "first", true),
			record(&calls, "second", false),
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)

		require.Size(t, 1, calls)
		assert.Equal(t, "first", calls[0])
	})

	t.Run("no interceptor", func(t *testing.T) {
		var calls []string

		handler := applyInterceptor(next(&calls))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)

		require.Size(t, 1, calls)
		assert.Equal(t, "handler", calls[0])
	})
}

func TestRouterHandler_AppliesCors(t *testing.T) {
	origin := "https://app.example.com"

	cors := NewCors().
		AllowedOrigins(origin).
		AllowedMethods("GET")

	handler := func(w http.ResponseWriter, req *http.Request, ctx *Context) result.Result {
		return result.JsonOk(map[string]int{"a": 1})
	}

	router := NewRouterWithMux(http.NewServeMux()).
		Cors(cors).
		Route(http.MethodGet, handler, "/x")

	server := httptest.NewServer(router.Handler())
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/x", nil)
	assert.Nil(t, err)
	req.Header.Set("Origin", origin)

	res, err := http.DefaultClient.Do(req)
	assert.Nil(t, err)

	assert.Equal(t, origin, res.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

	assert.Nil(t, res.Body.Close())
}
