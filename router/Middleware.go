package router

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type middleware func(w http.ResponseWriter, req *http.Request) bool

func corsMiddleware(cors Cors) middleware {
	return func(w http.ResponseWriter, req *http.Request) bool {
		w.Header().Add("Vary", "Origin")

		origin := req.Header.Get("Origin")
		if origin == "" {
			return false
		}

		allowOrigin, ok := cors.allowOrigin(origin)
		if !ok {
			return false
		}

		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)

		if cors.allowCredentials && allowOrigin != "*" {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		requestMethod := req.Header.Get("Access-Control-Request-Method")
		if req.Method != http.MethodOptions || requestMethod == "" {
			return false
		}

		w.Header().Set("Access-Control-Allow-Methods", strings.Join(cors.allowedMethods, ", "))
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(cors.allowedHeaders, ", "))

		w.WriteHeader(http.StatusNoContent)

		return true
	}
}

func httpsRedirectMiddleware(hostTLS int) middleware {
	portTLS := fmt.Sprintf(":%d", hostTLS)

	return func(w http.ResponseWriter, req *http.Request) bool {
		if req.TLS != nil {
			return false
		}

		host := req.Host
		if h, _, err := net.SplitHostPort(req.Host); err == nil {
			host = h
		}

		target := host
		if portTLS != "" && portTLS != "443" {
			target = net.JoinHostPort(host, portTLS)
		}

		url := fmt.Sprintf("https://%s%s", target, req.URL.RequestURI())
		http.Redirect(w, req, url, http.StatusPermanentRedirect)

		return true
	}
}

func applyMiddleware(next http.Handler, handlers []middleware) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		for _, handler := range handlers {
			if exit := handler(w, req); exit {
				return
			}
		}
		next.ServeHTTP(w, req)
	})
}
