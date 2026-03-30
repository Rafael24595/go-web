package support

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
)

type TestProduct struct {
	ID    int    `xml:"id"`
	Name  string `xml:"name"`
	Price string `xml:"price"`
}

type TestUser struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func NewRequest(data string) *http.Request {
	return &http.Request{
		Body: io.NopCloser(bytes.NewReader([]byte(data))),
	}
}

func NewRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}
