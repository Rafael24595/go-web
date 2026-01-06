package router_test

import (
	"testing"

	"github.com/Rafael24595/go-web/router"
)

func TestInputBytesOffline(t *testing.T) {
	req := newRequest("test data")

	data, res := router.InputBytes(req)
	if res != nil {
		t.Fatalf("unexpected error: %v", res)
	}
	if string(data) != "test data" {
		t.Fatalf("expected 'test data', got %q", string(data))
	}
}

func TestInputBytes(t *testing.T) {
	req := newRequest("hello world")

	data, res := router.InputBytes(req)
	if res != nil {
		t.Fatalf("unexpected error: %v", res)
	}
	if string(data) != "hello world" {
		t.Fatalf("expected %q, got %q", "hello world", string(data))
	}
}

func TestInputBytesWithOpts_LaxLimit(t *testing.T) {
	req := newRequest("1234567890")
	w := newRecorder()
	opts := router.InputOpts{Limit: 5, Strict: false}

	data, res := router.InputBytesWithOpts(w, req, opts)
	if res != nil {
		t.Fatalf("unexpected error: %v", res)
	}
	if string(data) != "12345" {
		t.Fatalf("expected '12345', got %q", string(data))
	}
}

func TestInputBytesWithOpts_StrictLimit(t *testing.T) {
	req := newRequest("1234567890")
	w := newRecorder()
	opts := router.InputOpts{Limit: 5, Strict: true}

	data, res := router.InputBytesWithOpts(w, req, opts)
	if res == nil {
		t.Fatal("expected error due to strict limit, got nil")
	}
	if len(data) > 5 {
		t.Fatalf("read %d bytes, expected at most 5", len(data))
	}
}

func TestInputText(t *testing.T) {
	req := newRequest("hello text")
	text, res := router.InputText(req)
	if res != nil {
		t.Fatalf("unexpected error: %v", res)
	}
	if text != "hello text" {
		t.Fatalf("expected 'hello text', got %q", text)
	}
}

func TestInputTextWithOpts(t *testing.T) {
	req := newRequest("abcdef")
	w := newRecorder()
	opts := router.InputOpts{Limit: 3, Strict: false}

	text, res := router.InputTextWithOpts(w, req, opts)
	if res != nil {
		t.Fatalf("unexpected error: %v", res)
	}
	if text != "abc" {
		t.Fatalf("expected 'abc', got %q", text)
	}
}

func TestInputJson(t *testing.T) {
	req := newRequest(`{"name":"Alice","age":30}`)

	user, res := router.InputJson[testUser](req)
	if res != nil {
		t.Fatalf("unexpected error: %v", res)
	}
	if user.Name != "Alice" || user.Age != 30 {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestInputJson_Invalid(t *testing.T) {
	req := newRequest(`{"name":"Bob",age:30}`)

	_, res := router.InputJson[testUser](req)
	if res == nil {
		t.Fatal("expected JSON decode error, got nil")
	}
}

func TestInputJsonWithOpts_StrictLimit(t *testing.T) {
	req := newRequest(`{"name":"Alice","age":30}`)
	w := newRecorder()
	opts := router.InputOpts{Limit: 5, Strict: true}

	_, res := router.InputJsonWithOpts[testUser](w, req, opts)
	if res == nil {
		t.Fatal("expected error due to strict limit, got nil")
	}
}

func TestInputXml(t *testing.T) {
	req := newRequest(`<Product><id>1</id><name>Book</name><price>12.5</price></Product>`)

	product, res := router.InputXml[testProduct](req)
	if res != nil {
		t.Fatalf("unexpected error: %v", res)
	}
	if product.ID != 1 || product.Name != "Book" || product.Price != "12.5" {
		t.Fatalf("unexpected product: %+v", product)
	}
}

func TestInputXml_Invalid(t *testing.T) {
	req := newRequest(`<Product><id>1</id><name>Book</name><price>12.5`)

	_, res := router.InputXml[testProduct](req)
	if res == nil {
		t.Fatal("expected XML decode error, got nil")
	}
}

func TestInputXmlWithOpts_LaxLimit(t *testing.T) {
	req := newRequest(`<Product><id>1</id><name>Book</name><price>12.5</price></Product>`)
	w := newRecorder()
	opts := router.InputOpts{Limit: 10, Strict: false}

	_, res := router.InputXmlWithOpts[testProduct](w, req, opts)
	if res == nil {
		t.Fatal("expected error due to strict limit, got nil")
	}
}
