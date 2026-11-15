package router

import (
	"net/http"
	"testing"
	"time"
)

func TestWithReadTimeout(t *testing.T) {
	server := &http.Server{}
	timeout := 5 * time.Second

	WithReadTimeout(timeout)(server)

	if server.ReadTimeout != timeout {
		t.Errorf("ReadTimeout = %v, want %v", server.ReadTimeout, timeout)
	}
}

func TestWithReadHeaderTimeout(t *testing.T) {
	server := &http.Server{}
	timeout := 3 * time.Second

	WithReadHeaderTimeout(timeout)(server)

	if server.ReadHeaderTimeout != timeout {
		t.Errorf("ReadHeaderTimeout = %v, want %v", server.ReadHeaderTimeout, timeout)
	}
}

func TestWithWriteTimeout(t *testing.T) {
	server := &http.Server{}
	timeout := 10 * time.Second

	WithWriteTimeout(timeout)(server)

	if server.WriteTimeout != timeout {
		t.Errorf("WriteTimeout = %v, want %v", server.WriteTimeout, timeout)
	}
}

func TestWithIdleTimeout(t *testing.T) {
	server := &http.Server{}
	timeout := 30 * time.Second

	WithIdleTimeout(timeout)(server)

	if server.IdleTimeout != timeout {
		t.Errorf("IdleTimeout = %v, want %v", server.IdleTimeout, timeout)
	}
}

func TestWithMaxHeaderBytes(t *testing.T) {
	server := &http.Server{}
	size := 1 << 20

	WithMaxHeaderBytes(size)(server)

	if server.MaxHeaderBytes != size {
		t.Errorf("MaxHeaderBytes = %d, want %d", server.MaxHeaderBytes, size)
	}
}
