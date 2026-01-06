package router_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
)

type testProduct struct {
	ID    int    `xml:"id"`
	Name  string `xml:"name"`
	Price string `xml:"price"`
}

type testUser struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func newRequest(data string) *http.Request {
	return &http.Request{
		Body: io.NopCloser(bytes.NewReader([]byte(data))),
	}
}

func newRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}
