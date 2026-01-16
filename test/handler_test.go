package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rafael24595/go-web/router"
	"github.com/Rafael24595/go-web/router/result"
)

func okHandler(called *int) router.RequestHandler {
	return func(w http.ResponseWriter, r *http.Request, ctx *router.Context) result.Result {
		*called++
		return result.Ok(ctx)
	}
}

func errHandler(status int, called *int) router.RequestHandler {
	return func(w http.ResponseWriter, r *http.Request, ctx *router.Context) result.Result {
		*called++
		return result.Err(status, http.ErrAbortHandler)
	}
}

func newTestReq() (*httptest.ResponseRecorder, *http.Request, *router.Context) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := router.NewContext()
	return w, r, ctx
}

func TestFallbackHandlers_Empty(t *testing.T) {
	h := router.FallbackHandlers()

	w, r, ctx := newTestReq()
	res := h(w, r, ctx)

	if !res.Ok() {
		t.Fatalf("expected Next result when no handlers are provided")
	}
}

func TestFallbackHandlers_FirstOkStopsExecution(t *testing.T) {
	c1, c2 := 0, 0

	h := router.FallbackHandlers(
		okHandler(&c1),
		okHandler(&c2),
	)

	w, r, ctx := newTestReq()
	res := h(w, r, ctx)

	if !res.Ok() {
		t.Fatalf("expected Ok result")
	}

	if c1 != 1 {
		t.Fatalf("expected first handler to be called once")
	}

	if c2 != 0 {
		t.Fatalf("expected second handler NOT to be called")
	}
}

func TestFallbackHandlers_FallbackToSecond(t *testing.T) {
	c1, c2 := 0, 0

	h := router.FallbackHandlers(
		errHandler(401, &c1),
		okHandler(&c2),
	)

	w, r, ctx := newTestReq()
	res := h(w, r, ctx)

	if !res.Ok() {
		t.Fatalf("expected Ok result from second handler")
	}

	if c1 != 1 || c2 != 1 {
		t.Fatalf("expected both handlers to be executed")
	}
}

func TestFallbackHandlers_AllFail(t *testing.T) {
	c1, c2 := 0, 0

	h := router.FallbackHandlers(
		errHandler(401, &c1),
		errHandler(403, &c2),
	)

	w, r, ctx := newTestReq()
	res := h(w, r, ctx)

	if !res.Err() {
		t.Fatalf("expected Err result")
	}

	if res.Status() != 403 {
		t.Fatalf("expected last handler error to be returned")
	}

	if c1 != 1 && c1 == c2 {
		t.Fatalf("expected both handlers to be executed")
	}
}

func TestValidateHandlers_Empty(t *testing.T) {
	h := router.ValidateHandlers()

	w, r, ctx := newTestReq()
	res := h(w, r, ctx)

	if !res.Ok() {
		t.Fatalf("expected Ok result when no handlers are provided")
	}
}

func TestValidateHandlers_AllOk(t *testing.T) {
	c1, c2 := 0, 0

	h := router.ValidateHandlers(
		okHandler(&c1),
		okHandler(&c2),
	)

	w, r, ctx := newTestReq()
	res := h(w, r, ctx)

	if !res.Ok() {
		t.Fatalf("expected Ok result")
	}

	if c1 != 1 && c1 == c2 {
		t.Fatalf("expected all handlers to be executed")
	}
}

func TestValidateHandlers_FailFast(t *testing.T) {
	c1, c2 := 0, 0

	h := router.ValidateHandlers(
		errHandler(401, &c1),
		okHandler(&c2),
	)

	w, r, ctx := newTestReq()
	res := h(w, r, ctx)

	if !res.Err() {
		t.Fatalf("expected Err result")
	}

	if c1 != 1 {
		t.Fatalf("expected first handler to be executed")
	}

	if c2 != 0 {
		t.Fatalf("expected second handler NOT to be executed")
	}
}

func TestCombinedHandlers(t *testing.T) {
	c1, c2, c3 := 0, 0, 0

	lax := router.FallbackHandlers(
		errHandler(401, &c1),
		okHandler(&c2),
	)

	strict := router.ValidateHandlers(
		lax,
		okHandler(&c3),
	)

	w, r, ctx := newTestReq()
	res := strict(w, r, ctx)

	if !res.Ok() {
		t.Fatalf("expected Ok result")
	}

	if c1 != 1 && c1 == c2 && c2 == c3 {
		t.Fatalf("unexpected handler execution counts")
	}
}
