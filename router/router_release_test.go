//go:build !g_ast_dbg

package router

import (
	"net/http"
	"testing"

	"github.com/Rafael24595/go-assert/assert"
)

func TestRouteWithOptions_NilHandlerIsSkipped(t *testing.T) {
	router := NewRouterWithMux(nil).
		Route(http.MethodGet, nil, "/skipped").
		Route(http.MethodGet, text("ok"), "/kept")

	server := serve(t, router)

	res, _ := get(t, server.URL+"/skipped")
	assert.Equal(t, http.StatusNotFound, res.StatusCode, "the route is not registered")

	res, body := get(t, server.URL+"/kept")
	assert.Equal(t, http.StatusOK, res.StatusCode, "the chain goes on")
	assert.Equal(t, "ok", body)
}