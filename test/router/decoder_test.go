package router_test

import (
	"testing"

	"github.com/Rafael24595/go-web/router"
	"github.com/Rafael24595/go-web/test/support"

	assert "github.com/Rafael24595/go-assert/assert/test"
)

func TestInputBytesOffline(t *testing.T) {
	req := support.NewRequest("test data")

	data, res := router.InputBytes(req)

	assert.Nil(t, res)
	assert.Equal(t, "test data", string(data))
}

func TestInputBytes(t *testing.T) {
	req := support.NewRequest("hello world")

	data, res := router.InputBytes(req)

	assert.Nil(t, res)
	assert.Equal(t, "hello world", string(data))
}

func TestInputBytesWithOpts_LaxLimit(t *testing.T) {
	req := support.NewRequest("1234567890")
	w := support.NewRecorder()
	opts := router.InputOpts{Limit: 5, Strict: false}

	data, res := router.InputBytesWithOpts(w, req, opts)

	assert.Nil(t, res)
	assert.Equal(t, "12345", string(data))
}

func TestInputBytesWithOpts_StrictLimit(t *testing.T) {
	req := support.NewRequest("1234567890")
	w := support.NewRecorder()
	opts := router.InputOpts{Limit: 5, Strict: true}

	data, res := router.InputBytesWithOpts(w, req, opts)

	assert.NotNil(t, res)
	assert.Len(t, 5, data)
}

func TestInputText(t *testing.T) {
	req := support.NewRequest("hello text")
	text, res := router.InputText(req)

	assert.Nil(t, res)
	assert.Equal(t, "hello text", text)
}

func TestInputTextWithOpts(t *testing.T) {
	req := support.NewRequest("abcdef")
	w := support.NewRecorder()
	opts := router.InputOpts{Limit: 3, Strict: false}

	text, res := router.InputTextWithOpts(w, req, opts)

	assert.Nil(t, res)
	assert.Equal(t, "abc", text)
}

func TestInputJson(t *testing.T) {
	req := support.NewRequest(`{"name":"Rafael","age":30}`)

	user, res := router.InputJson[support.TestUser](req)

	assert.Nil(t, res)
	assert.Equal(t, "Rafael", user.Name)
	assert.Equal(t, 30, user.Age)
}

func TestInputJson_Invalid(t *testing.T) {
	req := support.NewRequest(`{"name":"Bob",age:30}`)

	_, res := router.InputJson[support.TestUser](req)
	assert.NotNil(t, res)
}

func TestInputJsonWithOpts_StrictLimit(t *testing.T) {
	req := support.NewRequest(`{"name":"Alice","age":30}`)
	w := support.NewRecorder()
	opts := router.InputOpts{Limit: 5, Strict: true}

	_, res := router.InputJsonWithOpts[support.TestUser](w, req, opts)
	assert.NotNil(t, res)
}

func TestInputXml(t *testing.T) {
	req := support.NewRequest(`<Product><id>1</id><name>Book</name><price>12.5</price></Product>`)

	product, res := router.InputXml[support.TestProduct](req)

	assert.Nil(t, res)
	assert.Equal(t, 1, product.ID)
	assert.Equal(t, "Book", product.Name)
	assert.Equal(t, "12.5", product.Price)
}

func TestInputXml_Invalid(t *testing.T) {
	req := support.NewRequest(`<Product><id>1</id><name>Book</name><price>12.5`)

	_, res := router.InputXml[support.TestProduct](req)
	assert.NotNil(t, res)
}

func TestInputXmlWithOpts_LaxLimit(t *testing.T) {
	req := support.NewRequest(`<Product><id>1</id><name>Book</name><price>12.5</price></Product>`)
	w := support.NewRecorder()
	opts := router.InputOpts{Limit: 10, Strict: false}

	_, res := router.InputXmlWithOpts[support.TestProduct](w, req, opts)
	assert.NotNil(t, res)
}
