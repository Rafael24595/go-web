package result

import (
	"net/http"
)

// Result represents the outcome of a route handler execution.
//
// It encapsulates:
//   - Whether the operation succeeded (`isOk`)
//   - HTTP status code (`status`)
//   - Response payload (`payload`)
//   - Encoder (`encoder`) to format the response
type Result struct {
	isOk    bool
	status  int
	payload any
	encoder ResultEncoder
}

// Ok returns a successful plain-text result with HTTP 200.
func Ok(payload any) Result {
	return Result{
		isOk:    true,
		status:  http.StatusOK,
		payload: payload,
		encoder: NewTextEncoder(),
	}
}

// JsonOk returns a successful JSON result with HTTP 200.
func JsonOk(payload any) Result {
	return Result{
		isOk:    true,
		status:  http.StatusOK,
		payload: payload,
		encoder: NewJsonEncoder(),
	}
}

// XmlOk returns a successful XML result with HTTP 200.
func XmlOk(payload any) Result {
	return Result{
		isOk:    true,
		status:  http.StatusOK,
		payload: payload,
		encoder: NewXmlEncoder(),
	}
}

// CustomOk returns a successful result with HTTP 200
// using a custom encoder.
func CustomOk(payload any, encoder ResultEncoder) Result {
	return Result{
		isOk:    true,
		status:  http.StatusOK,
		payload: payload,
		encoder: encoder,
	}
}

// Oks returns a successful plain-text result
// with a custom HTTP status.
func Oks(status int, payload any) Result {
	return Result{
		isOk:    true,
		status:  status,
		payload: payload,
		encoder: NewTextEncoder(),
	}
}

// JsonOks returns a successful JSON result
// with a custom HTTP status.
func JsonOks(status int, payload any) Result {
	return Result{
		isOk:    true,
		status:  status,
		payload: payload,
		encoder: NewJsonEncoder(),
	}
}

// XmlOks returns a successful XML result
// with a custom HTTP status.
func XmlOks(status int, payload any) Result {
	return Result{
		isOk:    true,
		status:  status,
		payload: payload,
		encoder: NewXmlEncoder(),
	}
}

// CustomOks returns a successful result
// with a custom HTTP status and encoder.
func CustomOks(status int, payload any, encoder ResultEncoder) Result {
	return Result{
		isOk:    true,
		status:  status,
		payload: payload,
		encoder: encoder,
	}
}

// Err returns a plain-text error result with a given HTTP status.
func Err(status int, err error) Result {
	message := ""
	if err != nil {
		message = err.Error()
	}

	return Result{
		isOk:    false,
		status:  status,
		payload: message,
		encoder: NewTextEncoder(),
	}
}

// JsonErr returns an error response encoded as JSON.
func JsonErr(status int, payload any) Result {
	return Result{
		isOk:    true,
		status:  status,
		payload: payload,
		encoder: NewJsonEncoder(),
	}
}

// XmlErr returns an error response encoded as XML.
func XmlErr(status int, payload any) Result {
	return Result{
		isOk:    true,
		status:  status,
		payload: payload,
		encoder: NewXmlEncoder(),
	}
}

// CustomErr returns an error response with a custom encoder.
func CustomErr(status int, payload any, encoder ResultEncoder) Result {
	return Result{
		isOk:    true,
		status:  status,
		payload: payload,
		encoder: encoder,
	}
}

// Accept returns a success result with the given status and no payload.
func Accept(status int) Result {
	return Oks(status, nil)
}

// Reject returns a plain-text error with the given status and no message.
func Reject(status int) Result {
	return Err(status, nil)
}

// Status returns the HTTP status code associated with the Result.
func (r Result) Status() int {
	return r.status
}

// Encoder returns the encoder associated with the Result.
func (r Result) Encoder() ResultEncoder {
	return r.encoder
}

// Payload returns the payload of the Result.
func (r Result) Payload() any {
	return r.payload
}

// Ok returns true if the result represents a successful operation.
func (r Result) Ok() bool {
	return r.isOk
}

// Err returns true if the result represents a failure.
func (r Result) Err() bool {
	return !r.isOk
}
