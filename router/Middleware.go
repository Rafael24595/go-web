package router

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type middleware func(w http.ResponseWriter, req *http.Request) bool

func corsMiddleware(cors *Cors) middleware {
	return func (w http.ResponseWriter, req *http.Request) bool {
	origin := strings.Join(cors.allowedOrigins, ", ")

	if origin == "*" {
		origin = req.Header.Get("Origin")
		w.Header().Set("Vary", "Origin")
	}

	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(cors.allowedMethods, ", "))
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(cors.allowedHeaders, ", "))
	w.Header().Set("Access-Control-Allow-Credentials", strconv.FormatBool(cors.allowCredentials))

	if req.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return true
	}

	return false
}
}

func httpsRedirectMiddleware(portTLS string) middleware {
	return func(w http.ResponseWriter, req *http.Request) bool {
		if req.TLS != nil {
			return false
		}

		host := req.Host
		if colon := strings.Index(host, ":"); colon != -1 {
			host = host[:colon]
		}
		url := fmt.Sprintf("https://%s%s%s", host, portTLS, req.RequestURI)
		http.Redirect(w, req, url, http.StatusMovedPermanently)

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
