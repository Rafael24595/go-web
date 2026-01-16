package router

import (
	"net/http"

	"github.com/Rafael24595/go-web/router/result"
)

// FallbackHandlers returns a RequestHandler that executes the given handlers
// sequentially until one of them returns an Ok result.
//
// Handlers are evaluated in order. Execution stops at the first handler that
// succeeds (Ok), and its result is returned. If none of the handlers succeed,
// the result of the last executed handler is returned. If no handlers are
// provided, the returned handler yields result.Next().
func FallbackHandlers(handlers ...RequestHandler) RequestHandler {
	return func(w http.ResponseWriter, r *http.Request, c *Context) result.Result {
		res := result.Next()
		for _, h := range handlers {
			res = h(w, r, c)
			if res.Ok() {
				return res
			}
		}
		return res
	}
}

// ValidateHandlers returns a RequestHandler that executes all given handlers
// sequentially and fails fast on the first error.
//
// Handlers are evaluated in order. If any handler returns an error (Err),
// execution stops and that result is returned immediately. If all handlers
// succeed, the returned handler yields an Ok result containing the context.
func ValidateHandlers(handlers ...RequestHandler) RequestHandler {
	return func(w http.ResponseWriter, r *http.Request, c *Context) result.Result {
		for _, h := range handlers {
			res := h(w, r, c)
			if res.Err() {
				return res
			}
		}
		return result.Ok(c)
	}
}
