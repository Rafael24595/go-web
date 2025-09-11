package result

import "net/http"

// Result represents the outcome of a route handler execution.
//
// It encapsulates:
//   - Whether the operation succeeded (`isOk`)
//   - HTTP status code (`status`)
//   - Optional response value (`ok`) for successful results
//   - Optional error (`err`) for failed results
type Result struct {
	isOk   bool
	status int
	ok     any
	err    error
}

// Ok creates a Result representing a successful response with HTTP 200 OK.
//
// Example:
//   return result.Ok(map[string]string{"message": "success"})
func Ok(ok any) Result {
	return Result{
		isOk:   true,
		status: http.StatusOK,
		ok:     ok,
		err:    nil,
	}
}

// Oks creates a Result representing a successful response with a custom status code.
//
// Example:
//   return result.Oks(http.StatusCreated, newUser)
func Oks(status int, ok any) Result {
	return Result{
		isOk:   true,
		status: status,
		ok:     ok,
		err:    nil,
	}
}

// Err creates a Result representing a failed response with a specific status code and error.
//
// Example:
//   return result.Err(http.StatusBadRequest, errors.New("invalid input"))
func Err(status int, err error) Result {
	return Result{
		isOk:   false,
		status: status,
		ok:     nil,
		err:    err,
	}
}

// Accept creates a Result representing a successful response with no body
// but a custom HTTP status code.
//
// Example:
//   return result.Accept(http.StatusAccepted)
func Accept(status int) Result {
	return Result{
		isOk:   true,
		status: status,
		ok:     nil,
		err:    nil,
	}
}

// Reject creates a Result representing a failed response with no body
// but a custom HTTP status code.
//
// Example:
//   return result.Reject(http.StatusForbidden)
func Reject(status int) Result {
	return Result{
		isOk:   false,
		status: status,
		ok:     nil,
		err:    nil,
	}
}

// Status returns the HTTP status code associated with the Result.
func (r Result) Status() int {
	return r.status
}

// Ok returns the successful response value (if any) and a boolean indicating success.
//
// If the Result represents an error, the boolean will be false.
func (r Result) Ok() (any, bool) {
	return r.ok, r.isOk
}

// Err returns the error (if any) and a boolean indicating failure.
//
// If the Result represents a success, the boolean will be false.
func (r Result) Err() (error, bool) {
	return r.err, !r.isOk
}
