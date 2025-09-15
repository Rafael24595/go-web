package router

// Cors represents the configuration for Cross-Origin Resource Sharing (CORS)
// on the Router.
//
// It defines which origins, HTTP methods, headers, and credentials are allowed
// when handling cross-origin requests.
type Cors struct {
	allowedOrigins   []string
	allowedMethods   []string
	allowedHeaders   []string
	allowCredentials bool
}

// EmptyCors creates a new Cors instance with all fields empty or disabled.
//
// By default, no origins, methods, or headers are allowed, and credentials
// are not permitted. Use the builder-style methods to configure it.
func EmptyCors() *Cors {
	return &Cors{
		allowedOrigins:   make([]string, 0),
		allowedMethods:   make([]string, 0),
		allowedHeaders:   make([]string, 0),
		allowCredentials: false,
	}
}

// PermissiveCors returns a Cors instance configured with a fully permissive policy.
//
// The configuration allows:
//   - Any origin (`*`)
//   - HTTP methods: GET, POST, PUT, DELETE, OPTIONS
//   - Headers: Content-Type, Authorization
//   - Credentials (cookies, HTTP auth)
//
// This is useful for development, testing, or internal services where
// strict CORS rules are not required. For production, it is generally
// recommended to configure CORS explicitly using EmptyCors() and the
// builder methods.
func PermissiveCors() *Cors {
	return EmptyCors().
		AllowedOrigins("*").
		AllowedMethods("GET", "POST", "PUT", "DELETE", "OPTIONS").
		AllowedHeaders("Content-Type", "Authorization").
		AllowCredentials()
}

// AllowedOrigins sets the list of allowed origins for CORS requests.
//
// Example:
//   cors := EmptyCors().AllowedOrigins("https://example.com", "https://api.example.com")
//
// Returns the Cors instance for fluent configuration.
func (c *Cors) AllowedOrigins(allowedOrigins ...string) *Cors {
	c.allowedOrigins = allowedOrigins
	return c
}

// AllowedMethods sets the list of allowed HTTP methods for CORS requests.
//
// Example:
//   cors := EmptyCors().AllowedMethods("GET", "POST", "PUT")
//
// Returns the Cors instance for fluent configuration.
func (c *Cors) AllowedMethods(allowedMethods ...string) *Cors {
	c.allowedMethods = allowedMethods
	return c
}

// AllowedHeaders sets the list of allowed HTTP headers for CORS requests.
//
// Example:
//   cors := EmptyCors().AllowedHeaders("Content-Type", "Authorization")
//
// Returns the Cors instance for fluent configuration.
func (c *Cors) AllowedHeaders(allowedHeaders ...string) *Cors {
	c.allowedHeaders = allowedHeaders
	return c
}

// AllowCredentials enables sending credentials (cookies, HTTP auth) in CORS requests.
//
// Returns the Cors instance for fluent configuration.
func (c *Cors) AllowCredentials() *Cors {
	c.allowCredentials = true
	return c
}

// NotAllowCredentials disables sending credentials in CORS requests.
//
// Returns the Cors instance for fluent configuration.
func (c *Cors) NotAllowCredentials() *Cors {
	c.allowCredentials = false
	return c
}

// IsEmpty returns true if the CORS configuration has no allowed
// origins, methods, or headers defined.
//
// It can be used to check whether the Cors instance is unconfigured.
func (c *Cors) IsEmpty() bool {
	return len(c.allowedOrigins) == 0 &&
		len(c.allowedMethods) == 0 &&
		len(c.allowedHeaders) == 0
}

// IsNotEmpty returns true if the CORS configuration is not empty.
// This is equivalent to !IsEmpty().
//
// It can be used to verify that the Cors instance has at least one
// allowed origin, method, or header defined.
func (c *Cors) IsNotEmpty() bool {
	return !c.IsEmpty()
}
