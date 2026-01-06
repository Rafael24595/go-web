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

// InputOpts defines options for reading HTTP request bodies.
//
// It allows controlling the maximum number of bytes to read
// and whether to enforce a strict limit. These options are
// used by the `WithOpts` variants of the input functions.
//
// Fields:
//   - Limit: maximum number of bytes to read from the request body.
//            Set to 0 for no limit.
//   - Strict: if true, reading more than Limit bytes will return an error.
//             If false, the reader will return only up to Limit bytes without error.
//
// Example:
//
//   opts := router.InputOpts{
//       Limit: 1024 * 1024, // 1 MB
//       Strict: true,       // error if exceeded
//   }
//   data, res := router.InputBytesWithOpts(w, r, opts)
type InputOpts struct {
	Strict bool  // whether to enforce the limit strictly
	Limit  int64 // maximum number of bytes to read; 0 = unlimited
}

// InputBytes reads the entire request body as raw bytes.
//
// If reading the body is successful, it returns the content as a byte slice
// and a nil result. If an error occurs while reading the body, it returns
// an empty byte slice and a non-nil *result.Result with status
// 400 Bad Request. The request body is always closed by this function.
//
// Example:
//
//	func handler(w http.ResponseWriter, r *http.Request, ctx router.Context) result.Result {
//	    data, res := router.InputBytes(r)
//	    if res != nil {
//	        return *res
//	    }
//	    return result.BytesOk(data)
//	}
func InputBytes(r *http.Request) ([]byte, *result.Result) {
	defer r.Body.Close()

	raw, res := readAllBytes(r)
	if res != nil {
		return raw, res
	}
	return raw, nil
}

// InputText reads the entire request body as string.
//
// If reading the body is successful, it returns the content as a string
// and a nil result. If an error occurs while reading the body, it returns
// an empty byte slice and a non-nil *result.Result with status
// 400 Bad Request. The request body is always closed by this function.
//
// Example:
//
//	func handler(w http.ResponseWriter, r *http.Request, ctx router.Context) result.Result {
//	    data, res := router.InputText(r)
//	    if res != nil {
//	        return *res
//	    }
//	    return result.TextOk(data)
//	}
func InputText(r *http.Request) (string, *result.Result) {
	raw, res := InputBytes(r)
	return string(raw), res
}

// InputJson parses the request body as JSON into a value of type T.
//
// If decoding is successful, it returns the payload and a nil result.
// If an error occurs while reading or decoding the body, it returns
// the zero value of T and a non-nil *result.Result with status
// 422 Unprocessable Entity. The request body is always closed by this function.
//
// Example:
//
//	type User struct {
//	    Name string `json:"name"`
//	    Age  int    `json:"age"`
//	}
//
//	func handler(w http.ResponseWriter, r *http.Request, ctx router.Context) result.Result {
//	    user, res := router.InputJson[testUser](r)
//	    if res != nil {
//	        return *res
//	    }
//	    return result.JsonOk(user)
//	}
func InputJson[T any](r *http.Request) (T, *result.Result) {
	raw, res := InputBytes(r)
	if res != nil {
		var zero T
		return zero, res
	}
	return jsonDecode[T](raw)
}

// InputXml parses the request body as XML into a value of type T.
//
// The decoder supports multiple character sets via
// golang.org/x/net/html/charset.
//
// If decoding is successful, it returns the payload and a nil result.
// If an error occurs while reading or decoding the body, it returns
// the zero value of T and a non-nil *result.Result with status
// 422 Unprocessable Entity. The request body is always closed by this function.
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
//	    product, res := router.InputXml[testProduct](r)
//	    if res != nil {
//	        return *res
//	    }
//	    return result.XmlOk(product)
//	}
func InputXml[T any](r *http.Request) (T, *result.Result) {
	raw, res := InputBytes(r)
	if res != nil {
		var zero T
		return zero, res
	}
	return xmlDecode[T](raw)
}

// InputBytesWithOpts reads the entire request body as raw bytes with additional options.
//
// The request body is always closed by this function. Use InputOpts to specify:
//   - Limit: maximum number of bytes to read (0 means no limit)
//   - Strict: whether to return an error if the limit is exceeded
//
// If reading the body succeeds, it returns the content as a byte slice and nil result.
// If an error occurs (body too large in strict mode, or read error), it returns
// a non-nil *result.Result with the appropriate status (413 or 422).
//
// Example:
//
//	func handler(w http.ResponseWriter, r *http.Request, ctx router.Context) result.Result {
//	    opts := router.InputOpts{Limit: 1 << 20, Strict: true} // 1 MB strict limit
//	    data, res := router.InputBytesWithOpts(w, r, opts)
//	    if res != nil {
//	        return *res
//	    }
//	    return result.BytesOk(data)
//	}
func InputBytesWithOpts(w http.ResponseWriter, r *http.Request, opts InputOpts) ([]byte, *result.Result) {
	defer r.Body.Close()

	raw, res := readOptBytes(w, r, opts.Limit, opts.Strict)
	if res != nil {
		return raw, res
	}
	return raw, nil
}

// InputTextWithOpts reads the entire request body as string with additional options.
//
// The request body is always closed by this function. Use InputOpts to specify:
//   - Limit: maximum number of bytes to read (0 means no limit)
//   - Strict: whether to return an error if the limit is exceeded
//
// If reading the body succeeds, it returns the content as a byte slice and nil result.
// If an error occurs (body too large in strict mode, or read error), it returns
// a non-nil *result.Result with the appropriate status (413 or 422).
//
// Example:
//
//	func handler(w http.ResponseWriter, r *http.Request, ctx router.Context) result.Result {
//	    opts := router.InputOpts{Limit: 1 << 20, Strict: true} // 1 MB strict limit
//	    data, res := router.InputTextWithOpts(w, r, opts)
//	    if res != nil {
//	        return *res
//	    }
//	    return result.TextOk(data)
//	}
func InputTextWithOpts(w http.ResponseWriter, r *http.Request, opts InputOpts) (string, *result.Result) {
	raw, res := InputBytesWithOpts(w, r, opts)
	return string(raw), res
}

// InputJsonWithOpts reads the entire request body as JSON into a value of type T with additional options.
//
// The request body is always closed by this function. Use InputOpts to specify:
//   - Limit: maximum number of bytes to read (0 means no limit)
//   - Strict: whether to return an error if the limit is exceeded
//
// If reading the body succeeds, it returns the content as a byte slice and nil result.
// If an error occurs (body too large in strict mode, or read error), it returns
// a non-nil *result.Result with the appropriate status (413 or 422).
//
// Example:
//
//	type User struct {
//	    Name string `json:"name"`
//	    Age  int    `json:"age"`
//	}
//
//	func handler(w http.ResponseWriter, r *http.Request, ctx router.Context) result.Result {
//	    opts := router.InputOpts{Limit: 1 << 20, Strict: true} // 1 MB strict limit
//	    user, res := router.InputJsonWithOpts[testUser](w, r, opts)
//	    if res != nil {
//	        return *res
//	    }
//	    return result.JsonOk(user)
//	}
func InputJsonWithOpts[T any](w http.ResponseWriter, r *http.Request, opts InputOpts) (T, *result.Result) {
	raw, res := InputBytesWithOpts(w, r, opts)
	if res != nil {
		var zero T
		return zero, res
	}
	return jsonDecode[T](raw)
}

// InputXmlWithOpts reads the entire request body as JSON into a value of type T with additional options.
//
// The request body is always closed by this function. Use InputOpts to specify:
//   - Limit: maximum number of bytes to read (0 means no limit)
//   - Strict: whether to return an error if the limit is exceeded
//
// If reading the body succeeds, it returns the content as a byte slice and nil result.
// If an error occurs (body too large in strict mode, or read error), it returns
// a non-nil *result.Result with the appropriate status (413 or 422).
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
//	    opts := router.InputOpts{Limit: 1 << 20, Strict: true} // 1 MB strict limit
//	    product, res := router.InputXmlWithOpts[testProduct](w, r, opts)
//	    if res != nil {
//	        return *res
//	    }
//	    return result.XmlOk(product)
//	}
func InputXmlWithOpts[T any](w http.ResponseWriter, r *http.Request, opts InputOpts) (payload T, res *result.Result) {
	raw, res := InputBytesWithOpts(w, r, opts)
	if res != nil {
		var zero T
		return zero, res
	}
	return xmlDecode[T](raw)
}

func jsonDecode[T any](raw []byte) (T, *result.Result) {
	var payload T

	err := json.Unmarshal(raw, &payload)
	if err != nil {
		result := result.Err(http.StatusUnprocessableEntity, err)
		return payload, &result
	}

	return payload, nil
}

func xmlDecode[T any](raw []byte) (T, *result.Result) {
	var payload T

	decoder := xml.NewDecoder(bytes.NewReader(raw))
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(&payload); err != nil {
		result := result.Err(http.StatusUnprocessableEntity, err)
		return payload, &result
	}

	return payload, nil
}

func readOptBytes(w http.ResponseWriter, r *http.Request, limit int64, strict bool) ([]byte, *result.Result) {
	if limit <= 0 {
		return readAllBytes(r)
	}

	if strict {
		return readMaxBytes(w, r, limit)
	}

	return readLaxBytes(r, limit)
}

func readAllBytes(r *http.Request) ([]byte, *result.Result) {
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		result := result.Err(http.StatusUnprocessableEntity, err)
		return bytes, &result
	}
	return bytes, nil
}

func readMaxBytes(w http.ResponseWriter, r *http.Request, limit int64) ([]byte, *result.Result) {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		res := result.Err(http.StatusRequestEntityTooLarge)
		return data, &res
	}
	return data, nil
}

func readLaxBytes(r *http.Request, limit int64) ([]byte, *result.Result) {
	limited := io.LimitReader(r.Body, limit)
	data, err := io.ReadAll(limited)
	if err != nil {
		res := result.Err(http.StatusUnprocessableEntity, err)
		return data, &res
	}
	return data, nil
}
