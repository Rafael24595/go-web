package router

// Any wraps a value of any type and provides type-safe accessors.
type Any struct {
	item any
}

func anyFrom(item any) Any {
	return Any{
		item: item,
	}
}

// Bool attempts to cast the wrapped value to bool.
// Returns the value and true if successful, otherwise false and false.
func (a Any) Bool() (bool, bool) {
	if res, ok := a.item.(bool); ok {
		return res, true
	}
	return false, false
}

// Boold returns the wrapped bool value or the provided default if casting fails.
func (a Any) Boold(def bool) bool {
	if res, ok := a.Bool(); ok {
		return res
	}
	return def
}

// String attempts to cast the wrapped value to string.
// Returns the value and true if successful, otherwise "" and false.
func (a Any) String() (string, bool) {
	if res, ok := a.item.(string); ok {
		return res, true
	}
	return "", false
}

// Stringd returns the wrapped string value or the provided default if casting fails.
func (a Any) Stringd(def string) string {
	if res, ok := a.String(); ok {
		return res
	}
	return def
}

// Int attempts to cast the wrapped value to int.
// Returns the value and true if successful, otherwise 0 and false.
func (a Any) Int() (int, bool) {
	if res, ok := a.item.(int); ok {
		return res, true
	}
	return 0, false
}

// Intd returns the wrapped int value or the provided default if casting fails.
func (a Any) Intd(def int) int {
	if res, ok := a.Int(); ok {
		return res
	}
	return def
}

// Int32 attempts to cast the wrapped value to int32.
// Returns the value and true if successful, otherwise 0 and false.
func (a Any) Int32() (int32, bool) {
	if res, ok := a.item.(int32); ok {
		return res, true
	}
	return 0, false
}

// Int32d returns the wrapped int32 value or the provided default if casting fails.
func (a Any) Int32d(def int32) int32 {
	if res, ok := a.Int32(); ok {
		return res
	}
	return def
}

// Int64 attempts to cast the wrapped value to int64.
// Returns the value and true if successful, otherwise 0 and false.
func (a Any) Int64() (int64, bool) {
	if res, ok := a.item.(int64); ok {
		return res, true
	}
	return 0, false
}

// Int64d returns the wrapped int64 value or the provided default if casting fails.
func (a Any) Int64d(def int64) int64 {
	if res, ok := a.Int64(); ok {
		return res
	}
	return def
}

// Float32 attempts to cast the wrapped value to float32.
// Returns the value and true if successful, otherwise 0 and false.
func (a Any) Float32() (float32, bool) {
	if res, ok := a.item.(float32); ok {
		return res, true
	}
	return 0, false
}

// Float32d returns the wrapped float32 value or the provided default if casting fails.
func (a Any) Float32d(def float32) float32 {
	if res, ok := a.Float32(); ok {
		return res
	}
	return def
}

// Float64 attempts to cast the wrapped value to float64.
// Returns the value and true if successful, otherwise 0 and false.
func (a Any) Float64() (float64, bool) {
	if res, ok := a.item.(float64); ok {
		return res, true
	}
	return 0, false
}

// Float64d returns the wrapped float64 value or the provided default if casting fails.
func (a Any) Float64d(def float64) float64 {
	if res, ok := a.Float64(); ok {
		return res
	}
	return def
}

// Str attempts to cast the wrapped value to a generic type T.
// Returns the value and true if successful, otherwise zero value and false.
func Str[T any](a Any) (T, bool) {
	if res, ok := a.item.(T); ok {
		return res, true
	}
	var zero T
	return zero, false
}

// Strd returns the wrapped value cast to T, or the provided default if casting fails.
func Strd[T any](a Any, def T) T {
	if res, ok := Str[T](a); ok {
		return res
	}
	return def
}
