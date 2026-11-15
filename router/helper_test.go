package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rafael24595/go-assert/assert"
	"github.com/Rafael24595/go-web/router/result"
)

func okHandler(called *int) RequestHandler {
	return func(w http.ResponseWriter, r *http.Request, ctx *Context) result.Result {
		*called++
		return result.Ok(ctx)
	}
}

func errHandler(status int, called *int) RequestHandler {
	return func(w http.ResponseWriter, r *http.Request, ctx *Context) result.Result {
		*called++
		return result.Err(status, http.ErrAbortHandler)
	}
}

func newTestReq() (*httptest.ResponseRecorder, *http.Request, *Context) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := NewContext()
	return w, r, ctx
}

func TestFallbackHandlers_Empty(t *testing.T) {
	h := FallbackHandlers()

	w, r, ctx := newTestReq()
	res := h(w, r, ctx)

	assert.True(t, res.Ok())
}

func TestFallbackHandlers_FirstOkStopsExecution(t *testing.T) {
	c1, c2 := 0, 0

	h := FallbackHandlers(
		okHandler(&c1),
		okHandler(&c2),
	)

	w, r, ctx := newTestReq()
	res := h(w, r, ctx)

	assert.True(t, res.Ok())

	assert.Equal(t, 1, c1)
	assert.Equal(t, 0, c2)
}

func TestFallbackHandlers_FallbackToSecond(t *testing.T) {
	c1, c2 := 0, 0

	h := FallbackHandlers(
		errHandler(401, &c1),
		okHandler(&c2),
	)

	w, r, ctx := newTestReq()
	res := h(w, r, ctx)

	assert.True(t, res.Ok())

	assert.Equal(t, 1, c1)
	assert.Equal(t, 1, c2)
}

func TestFallbackHandlers_AllFail(t *testing.T) {
	c1, c2 := 0, 0

	h := FallbackHandlers(
		errHandler(401, &c1),
		errHandler(403, &c2),
	)

	w, r, ctx := newTestReq()
	res := h(w, r, ctx)

	assert.True(t, res.Err())

	assert.Equal(t, 403, res.Status())
	assert.Equal(t, 1, c1)
	assert.Equal(t, 1, c2)
}

func TestValidateHandlers_Empty(t *testing.T) {
	h := ValidateHandlers()

	w, r, ctx := newTestReq()
	res := h(w, r, ctx)

	assert.True(t, res.Ok())
}

func TestValidateHandlers_AllOk(t *testing.T) {
	c1, c2 := 0, 0

	h := ValidateHandlers(
		okHandler(&c1),
		okHandler(&c2),
	)

	w, r, ctx := newTestReq()
	res := h(w, r, ctx)

	assert.True(t, res.Ok())

	assert.Equal(t, 1, c1)
	assert.Equal(t, 1, c2)
}

func TestValidateHandlers_FailFast(t *testing.T) {
	c1, c2 := 0, 0

	h := ValidateHandlers(
		errHandler(401, &c1),
		okHandler(&c2),
	)

	w, r, ctx := newTestReq()
	res := h(w, r, ctx)

	assert.True(t, res.Err())

	assert.Equal(t, 1, c1)
	assert.Equal(t, 0, c2)
}

func TestCombinedHandlers(t *testing.T) {
	c1, c2, c3 := 0, 0, 0

	lax := FallbackHandlers(
		errHandler(401, &c1),
		okHandler(&c2),
	)

	strict := ValidateHandlers(
		lax,
		okHandler(&c3),
	)

	w, r, ctx := newTestReq()
	res := strict(w, r, ctx)

	assert.True(t, res.Ok())

	assert.Equal(t, 1, c1)
	assert.Equal(t, 1, c2)
	assert.Equal(t, 1, c3)
}

func TestFallbackHandlers_JsonErrFallsBack(t *testing.T) {
	count1 := 0
	count2 := 0

	jsonErr := func(w http.ResponseWriter, r *http.Request, ctx *Context) result.Result {
		count1 += 1
		return result.JsonErr(http.StatusUnauthorized, map[string]string{"error": "no token"})
	}

	h := FallbackHandlers(jsonErr, okHandler(&count2))

	w, r, ctx := newTestReq()
	res := h(w, r, ctx)

	assert.True(t, res.Ok())
	assert.Equal(t, 1, count1)
	assert.Equal(t, 1, count2)
}
