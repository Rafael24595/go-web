package router

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"

	"github.com/Rafael24595/go-web/router/result"
	"golang.org/x/net/html/charset"
)

// InputText reads the entire request body as raw bytes.
//
// If reading the body is successful, it returns the content as a byte slice
// and a nil result. If an error occurs while reading the body, it returns
// an empty byte slice and a non-nil *result.Result with status
// 400 Bad Request.
//
// Example:
//
//	func handler(w http.ResponseWriter, r *http.Request, ctx router.Context) result.Result {
//	    data, res := router.InputText(r)
//	    if res != nil {
//	        return *res
//	    }
//	    return result.BytesOk(data)
//	}
func InputText(r *http.Request) ([]byte, *result.Result) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		result := result.Err(http.StatusBadRequest, err)
		return make([]byte, 0), &result
	}

	return bodyBytes, nil
}

// InputJson parses the request body as JSON into a value of type T.
//
// If decoding is successful, it returns the payload and a nil result.
// If an error occurs while reading or decoding the body, it returns
// the zero value of T and a non-nil *result.Result with status
// 422 Unprocessable Entity.
//
// Example:
//
//	type User struct {
//	    Name string `json:"name"`
//	    Age  int    `json:"age"`
//	}
//
//	func handler(w http.ResponseWriter, r *http.Request, ctx router.Context) result.Result {
//	    user, res := router.InputJson[User](r)
//	    if res != nil {
//	        return *res
//	    }
//	    return result.JsonOk(user)
//	}
func InputJson[T any](r *http.Request) (T, *result.Result) {
	var payload T

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		result := result.Err(http.StatusUnprocessableEntity, err)
		return payload, &result
	}

	err = json.Unmarshal(bodyBytes, &payload)
	if err != nil {
		result := result.Err(http.StatusUnprocessableEntity, err)
		return payload, &result
	}

	return payload, nil
}

// InputXml parses the request body as XML into a value of type T.
//
// The decoder supports multiple character sets via
// golang.org/x/net/html/charset.
//
// If decoding is successful, it returns the payload and a nil result.
// If an error occurs while reading or decoding the body, it returns
// the zero value of T and a non-nil *result.Result with status
// 422 Unprocessable Entity.
//
// Example:
//
//	type Product struct {
//	    ID    int    `xml:"id"`
//	    Name  string `xml:"name"`
//	    Price string `xml:"price"`
//	}
//
//	func handler(w http.ResponseWriter, r *http.Request, ctx router.Context) result.Result {
//	    product, res := router.InputXml[Product](r)
//	    if res != nil {
//	        return *res
//	    }
//	    return result.XmlOk(product)
//	}
func InputXml[T any](r *http.Request) (T, *result.Result) {
	var payload T

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		result := result.Err(http.StatusUnprocessableEntity, err)
		return payload, &result
	}

	decoder := xml.NewDecoder(bytes.NewReader(bodyBytes))
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(&payload); err != nil {
		result := result.Err(http.StatusUnprocessableEntity, err)
		return payload, &result
	}

	return payload, nil
}
