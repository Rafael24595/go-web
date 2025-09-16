package router

import (
	"iter"
	"maps"
)

// Context represents a key-value store where values are wrapped in Any.
type Context struct {
	ctx map[string]Any
}

// NewContext creates and returns a new empty Context.
func NewContext() *Context {
	return &Context{
		ctx: make(map[string]Any),
	}
}

// Get retrieves a value from the context by key.
// Returns a pointer to Any and true if the key exists, otherwise nil and false.
func (c *Context) Get(key string) (*Any, bool) {
	item, ok := c.ctx[key]
	return &item, ok
}

// Getz retrieves a value from the context by key.
// Returns the value wrapped in Any, or the zero value of Any if the key does not exist.
func (c *Context) Getz(key string) Any {
	return c.ctx[key]
}

// Put inserts or updates a value in the context.
// The value is automatically wrapped in Any.
func (c *Context) Put(key string, value any) *Context {
	c.ctx[key] = anyFrom(value)
	return c
}

// Delete removes a key from the context.
// Returns the deleted value and true if it existed, otherwise nil and false.
func (c *Context) Delete(key string) (*Any, bool) {
	item, ok := c.Get(key)
	delete(c.ctx, key)
	return item, ok
}

// Keys returns a sequence of all keys stored in the context.
func (c *Context) Keys(key string) iter.Seq[string] {
	return maps.Keys(c.ctx)
}

// Values returns a sequence of all values stored in the context.
func (c *Context) Values(key string) iter.Seq[Any] {
	return maps.Values(c.ctx)
}
