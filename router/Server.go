package router

import (
	"net/http"
	"time"
)

// ServerOption defines a function type that modifies an http.Server instance.
// This allows for flexible configuration of the server's properties.
// Addr, Handler, and ErrorLog are set by the Router and should not be modified by ServerOptions.
type ServerOption func(*http.Server)

// WithReadTimeout sets the ReadTimeout for the HTTP server.
func WithReadTimeout(timeout time.Duration) ServerOption {
	return func(server *http.Server) {
		server.ReadTimeout = timeout
	}
}

// WithReadHeaderTimeout sets the ReadHeaderTimeout for the HTTP server.
func WithReadHeaderTimeout(timeout time.Duration) ServerOption {
	return func(server *http.Server) {
		server.ReadHeaderTimeout = timeout
	}
}

// WithWriteTimeout sets the WriteTimeout for the HTTP server.
func WithWriteTimeout(timeout time.Duration) ServerOption {
	return func(server *http.Server) {
		server.WriteTimeout = timeout
	}
}

// WithIdleTimeout sets the IdleTimeout for the HTTP server.
func WithIdleTimeout(timeout time.Duration) ServerOption {
	return func(server *http.Server) {
		server.IdleTimeout = timeout
	}
}

// WithMaxHeaderBytes sets the MaxHeaderBytes for the HTTP server.
func WithMaxHeaderBytes(size int) ServerOption {
	return func(server *http.Server) {
		server.MaxHeaderBytes = size
	}
}
