//go:build g_ast_dbg && !g_ast_wrt

package router

import (
	"net/http"
	"testing"

	"github.com/Rafael24595/go-assert/assert"
)

func TestRouteWithOptions_NilHandlerPanicsInDebug(t *testing.T) {
	router := NewRouterWithMux(nil)

	assert.Panic(t, func() { router.Route(http.MethodGet, nil, "/a") })
	assert.Panic(t, func() { router.RouteWithOptions(http.MethodGet, nil, "/b") })
}