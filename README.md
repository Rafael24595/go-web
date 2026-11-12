# go-web

A lightweight and extensible Go web framework providing:

- Flexible HTTP routing and handlers
- CORS configuration
- Panic recovery and error handling
- Input deserialization for JSON and XML
- Result handling with multiple response formats
- Built-in support for JSON, XML, and plain text responses
- Documentation support (generic + Swagger/OpenAPI)

---

## Installation

```bash
go get github.com/Rafael24595/go-web
```

---

## Usage

### 1. Router

The `Router` is the central component that manages route registration, handlers, CORS, and documentation.

#### 1.0 Creating and serving a router

```go
route := router.NewRouter()                          // routes on http.DefaultServeMux
route := router.NewRouterWithMux(http.NewServeMux()) // routes on a mux of its own
```

`NewRouter()` registers its routes on the process-wide `http.DefaultServeMux`, so only one such router can declare a given route per process. Use `NewRouterWithMux` when several routers live in the same process, typically in tests. Passing `nil` creates a new mux.

The router can start its own server:

```go
route.Listen(":8080")                                                                   // every interface
route.Listen("127.0.0.1:8080")                                                          // local only
route.ListenTLS(":8443", router.TLSFromFiles("server.crt", "server.key"))
route.ListenWithTLS(":8080", ":8443", router.TLSFromFiles("server.crt", "server.key"))  // HTTP redirects to HTTPS
```

The certificate comes from a `TLSStrategy`: `TLSFromFiles` reads PEM files, `TLSFromConfig` takes a `*tls.Config` with certificates in memory (`Certificates`) or chosen per connection (`GetCertificate`, e.g. autocert or certificates renewed without restarting).

An address made only of digits is taken as a port (`"8080"` is `":8080"`). `ListenWithTLS` answers every plain HTTP request with a `308 Permanent Redirect` to the HTTPS address, keeping the method and the body; the host comes from the request and the port from the TLS address (omitted when it is 443).

Server settings are passed as options:

| Option | `http.Server` field |
|:--|:--|
| `router.WithReadTimeout(d)` | `ReadTimeout` |
| `router.WithReadHeaderTimeout(d)` | `ReadHeaderTimeout` |
| `router.WithWriteTimeout(d)` | `WriteTimeout` |
| `router.WithIdleTimeout(d)` | `IdleTimeout` |
| `router.WithMaxHeaderBytes(n)` | `MaxHeaderBytes` |

```go
route.Listen(":8080", router.WithReadHeaderTimeout(10*time.Second))
```

A `ServerOption` is a plain `func(*http.Server)`, so any other field can be set with your own function. `Addr`, `Handler` and `ErrorLog` are always set by the router and cannot be changed this way.

To stop the server gracefully, give the router a context. When it is done, the servers stop accepting connections, in-flight requests get up to the shutdown timeout to finish, and `Listen` returns `nil`. `OnListen` is called with the real address once the server is listening, before it serves anything:

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

err := router.NewRouter().
	Context(ctx).
	ShutdownTimeout(10 * time.Second). // default: router.DefaultShutdownTimeout (5s)
	OnListen(func(addr net.Addr) { 
        log.Printf("listening on %s", addr) 
    }).
	Listen("127.0.0.1:0") // port 0: OnListen tells which one was chosen
```

For tests (`httptest.NewServer`) or to mount the router inside a server of your own, `Handler()` returns it as an `http.Handler` with CORS applied.

#### 1.1 Basic route

```go
route.Route("GET", handler, "/hello")
```

#### 1.2 With parameters

```go
const PLACE = "place"

route.Route("GET", handler, "/hello/{%s}", PLACE)
```

#### 1.3 With options

```go
options := router.NewHandlerOptions(handler).
    Context(ctxHandler).
    Error(errHandler).
    Panic(panicHandler)

route.RouteWithOptions("GET", options, "/secure")
```

#### 1.4 With documentation

```go
const PLACE_DESC = "Place description"
const TEAPOT_DESC = "I'm a teapot"

doc := docs.DocRoute{
	Description: "description",
    Parameters: docs.DocOrderParameters{
		docs.Parameter(PLACE, PLACE_DESC),
	},
	Request: docs.DocJsonPayload[DtoPlace](),
	Responses: docs.DocResponses{
        "200": docs.DocXmlPayload[[]DtoGreetings](),
		"418": docs.DocText(TEAPOT_DESC),
	},
}

route.RouteDocument("GET", handler, "/hello/{%s}", doc)
```

#### 1.5 With documentation and options

```go
route.RouteDocumentWithOptions("GET", options, "/secure", doc)
```

#### 1.6 Context handler

```go
route.Contextualizer(func(http.ResponseWriter, *http.Request) (*router.Context, error) {
    context := router.NewContext()
    context.Put("username", "admin")
    return context, nil
})
```

#### 1.7 Error handler

```go
route.ErrorHandler(func(http.ResponseWriter, *http.Request, router.Context, result.Result) {
    if err, ok := result.Err(); ok && err != nil {
		http.Error(wrt, err.Error(), result.Status())
		return
	}

	wrt.WriteHeader(result.Status())
})
```

#### 1.8 Panic handler

```go
route.PanicHandler(func(w http.ResponseWriter, r *http.Request, rec any) {
    message := fmt.Sprintf("Recovered from panic: %v", r)
    http.Error(w, message, http.StatusInternalServerError)
})
```

#### 1.9 Context

Context is a key-value store for sharing data between handlers. All values are wrapped in `Any`.
The `Any` type holds a value of any kind and provides safe accessors to retrieve it as specific types.

Any methods:

- Bool() (bool, bool), Boold(def bool) bool
- String() (string, bool), Stringd(def string) string
- Int() (int, bool), Intd(def int) int
- Int32() (int32, bool), Int32d(def int32) int32
- Int64() (int64, bool), Int64d(def int64) int64
- Float32() (float32, bool), Float32d(def float32) float32
- Float64() (float64, bool), Float64d(def float64) float64
- Str[T any](a Any) (T, bool), Strd[T any](a Any, def T) T (generic type cast)

Usage:

```go
ctx := router.NewContext()

ctx.Put("username", "admin")
ctx.Put("userID", 123)

// user == "admin"
user := ctx.Getz("username").
    Stringd("anonymous")

// idAny == &Any{...} | ok == true
if idAny, ok := ctx.Get("userID"); ok {
    // id == 123 | ok == true
    id, ok := idAny.Int()
}

ctx.Delete("username")
```

Methods include:

- NewContext() *Context
- Get(key string) (*Any, bool)
- Getz(key string) Any
- Put(key string, value any) *Context
- Delete(key string) (*Any, bool)
- Keys(key string) iter.Seq[string]
- Values(key string) iter.Seq[Any]

#### 1.10 Handler composition

Handlers can be composed to build more complex request processing flows while keeping each handler focused on a single responsibility.

The router provides helper functions to combine multiple RequestHandler instances into a single handler with well-defined execution semantics.

#### 1.10.1 Fallback handlers

FallbackHandlers executes handlers sequentially until one of them succeeds.

- Handlers are evaluated in order
- Execution stops at the first handler that returns Ok
- If no handler succeeds, the last result is returned

This is useful for implementing fallback or alternative strategies, such as multiple authentication mechanisms.

```go
laxAuth := router.FallbackHandlers(
    authToken,
    authCookie,
)

route.GroupContextualizer(laxAuth, "user")
```

Typical use cases:

- Authentication fallback (token → cookie)
- Multiple resolvers for the same resource
- Optional features controlled by configuration

#### 1.10.2 Validation handlers

ValidateHandlers executes all handlers sequentially and fails fast on the first error.

- Handlers are evaluated in order
- Execution stops at the first handler that returns an error
- If all handlers succeed, the result is Ok

This is useful when all conditions must be satisfied before continuing.

```go
strictAuth := router.ValidateHandlers(
    laxAuth,
    requireVerifiedUser,
)

route.
    GroupContextualizer(strictAuth, "admin")
```


Typical use cases:

- Request validation pipelines
- Authorization and permission checks
- Precondition enforcement

#### 1.10.3 Combining strategies

Both composition strategies can be combined to express complex flows in a declarative way:

```go
laxAuth := router.FallbackHandlers(
    authToken,
    authCookie,
)

strictAuth := router.ValidateHandlers(
    laxAuth,
    requireVerifiedSession,
)
```

This approach keeps handlers simple and moves orchestration logic to the router configuration.

---

### 2. CORS

The `Cors` struct provides a fluent API for configuring cross-origin resource sharing.

```go
cors := router.EmptyCors().
    AllowedOrigins("https://example.com", "https://admin.example.com").
    AllowedMethods("GET", "POST").
    AllowedHeaders("Authorization").
    AllowCredentials()

route := router.NewRouter().Cors(cors)
```

How requests are answered:

- **No `Origin` header** (same-origin requests, `curl`, other servers): not a CORS request, no CORS headers.
- **A listed origin**: `Access-Control-Allow-Origin` echoes that origin alone, and `Access-Control-Allow-Credentials: true` is added when credentials are allowed.
- **An origin that is not listed**: no CORS headers; the request still reaches the route, and the browser blocks the response.
- **`*` in the list**: any origin is allowed. Without credentials the answer is the literal `*`. With credentials browsers refuse `*`, so the request origin is echoed instead, which lets every website send credentialed requests: list the origins explicitly when you need cookies.
- **Preflight** (`OPTIONS` with `Access-Control-Request-Method`) from an allowed origin: answered with `204 No Content` plus `Access-Control-Allow-Methods` and `Access-Control-Allow-Headers`; it never reaches the routes.
- Every response carries `Vary: Origin`, so caches keep a separate copy per origin.

#### 2.1 Permissive CORS

Allows any origin, with the usual methods and the `Content-Type` and `Authorization` headers, **without credentials**: a token sent in the `Authorization` header works, cookies from other origins do not.

```go
cors := router.PermissiveCors()
route := router.NewRouter().Cors(cors)
```

---

### 3. Input Deserializers

The framework provides built-in helper functions for deserializing request bodies into Go structs. These functions simplify handling JSON and XML payloads inside route handlers.

> **Note:** This methods always close the request body, ensuring proper resource management.

#### 3.1 Bytes Input

Reads the entire request body as raw bytes.
- On success: returns the raw bytes and nil.
- On failure: returns an empty vector of bytes and a *result.Result with status 400 Bad Request.

```go
func handler(w http.ResponseWriter, r *http.Request, ctx *router.Context) result.Result {
    bytes, res := router.InputBytes(r)
    if res != nil {
        return *res
    }

    return result.BytesOk(bytes)
}
```

#### 3.2 Text Input

Reads the entire request body as string.
- On success: returns the raw bytes and nil.
- On failure: returns an empty vector of bytes and a *result.Result with status 400 Bad Request.

```go
func handler(w http.ResponseWriter, r *http.Request, ctx *router.Context) result.Result {
    text, res := router.InputText(r)
    if res != nil {
        return *res
    }

    return result.TextOk(bytes)
}
```

#### 3.3 JSON Input

Parses the request body as JSON into a value of type T.
- On success: returns the parsed payload and nil.
- On failure: returns the zero value of T and a *result.Result with status 422 Unprocessable Entity.

```go
type User struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

func handler(w http.ResponseWriter, r *http.Request, ctx *router.Context) result.Result {
    user, res := router.InputJson[User](r)
    if res != nil {
        return *res
    }

    return result.JsonOk(map[string]string{
        "message": "Hello " + user.Name,
    })
}
```

#### 3.4 XML Input

Parses the request body as XML into a value of type T. The decoder supports multiple character sets via golang.org/x/net/html/charset.
- On success: returns the parsed payload and nil.
- On failure: returns the zero value of T and a *result.Result with status 422 Unprocessable Entity.

```go
type Product struct {
    ID    int    `xml:"id"`
    Name  string `xml:"name"`
    Price string `xml:"price"`
}

func handler(w http.ResponseWriter, r *http.Request, ctx *router.Context) result.Result {
    product, res := router.InputXml[Product](r)
    if res != nil {
        return *res
    }

    return result.XmlOk(product)
}
```

#### 3.5 Input With Options (WithOpts)

All input types have a WithOpts variant that allows you to control:
- Limit: maximum number of bytes to read (0 = unlimited).
- Strict: whether to return an error if the limit is exceeded.

```go
func handler(w http.ResponseWriter, r *http.Request, ctx *router.Context) result.Result {
    opts := router.InputOpts{Limit: 1 << 20, Strict: true} // 1 MB strict limit
    data, res := router.InputBytesWithOpts(w, r, opts)
    if res != nil {
        return *res
    }
    return result.BytesOk(data)
}
```
---

### 4. Result Handling

The `Result` type standardizes how responses are returned from route handlers.

#### 4.1 Custom Encoders

You can create your own response format by implementing the `ResultEncoder` interface. This allows full control over how the payload is serialized and which headers are returned.

#### Example: Uppercase Text Encoder

```go
// CustomEncoder implements the ResultEncoder interface
type CustomEncoder struct{}

// Encode converts the payload to a custom string format
func (c *CustomEncoder) Encode(payload any) ([]byte, error) {
	// Example: convert any payload to uppercase string
	payloadStr := fmt.Sprintf("%v", payload)
	payloadStr = strings.ToUpper(payloadStr)
	return []byte(payloadStr), nil
}

// Headers returns custom headers for the response
func (c *CustomEncoder) Headers() map[string]string {
	return map[string]string{
		"Content-Type": "text/custom",
	}
}
```

#### 4.2 Success Responses (200 OK by default)
- `Ok(payload)` → Successful response (`200 OK`) from any with plain text encoder.
- `TextOk(string)` → Successful response (`200 OK`) from string with plain text encoder.
- `JsonOk(payload)` → Successful response (`200 OK`) encoded as JSON.
- `XmlOk(payload)` → Successful response (`200 OK`) encoded as XML.
- `FileOk(payload)` → Successful response (`200 OK`) with a file encoder.
- `CustomOk(payload, encoder)` → Successful response (`200 OK`) with a custom encoder.

```go
// Plain text from any
return result.Ok([]string{"Operation successful"})

// Plain text from string
return result.TextOk("Operation successful")

// JSON
payload := map[string]string{"message": "Operation successful"}
return result.JsonOk(payload)

// XML
return result.XmlOk(struct {
    Message string `xml:"message"`
}{"Operation successful"})

// File
return result.FileOk("index.html")

// Custom encoder
return result.CustomOk("Custom response", &CustomEncoder{})
```

#### 4.3 Success Responses with Custom HTTP Status
- `Oks(status, payload)` → Successful response with a custom HTTP status code and plain text encoder.
- `JsonOks(status, payload)` → Successful response with a custom HTTP status code, encoded as JSON.
- `XmlOks(status, payload)` → Successful response with a custom HTTP status code, encoded as XML.
- `CustomOks(status, payload, encoder)` → Successful response with a custom HTTP status code and encoder.

```go
// Plain text with status 201 Created
return result.Oks(201, "Resource created")

// JSON with status 202 Accepted
return result.JsonOks(202, map[string]string{"status": "accepted"})

// XML with status 204 No Content
return result.XmlOks(204, struct{}{})

// Custom encoder with status 200 OK
return result.CustomOks(200, myPayload, &CustomEncoder{})
```

#### 4.4 Error Responses
- `Err(status, error)` → Error response with a specific HTTP status code and error message, plain text by default.
- `TextErr(status, string)` → Error response with a specific HTTP status code and message, plain text by default.
- `JsonErr(status, payload)` → Error response with a specific HTTP status code, encoded as JSON.
- `XmlErr(status, payload)` → Error response with a specific HTTP status code, encoded as XML.
- `CustomErr(status, payload, encoder)` → Error response with a specific HTTP status code and a custom encoder.

```go
// Plain text from error
return result.Err(400, errors.New("Bad request"))

// Plain text from string
return result.TextErr(400, "Bad request")

// JSON error
return result.JsonErr(404, map[string]string{"error": "Not found"})

// XML error
return result.XmlErr(500, struct {
    Message string `xml:"message"`
}{"Internal server error"})

// Custom encoder for errors
return result.CustomErr(403, "Forbidden", &CustomEncoder{})
```

#### 4.5 Responses without payload
- `Accept(status)` → Accept response with no payload (success) and custom HTTP status code.
- `Reject(status)` → Reject response with no payload (failure) and custom HTTP status code.

```go
// Accept with status 202
return result.Accept(202)

// Reject with status 403
return result.Reject(403)
```

#### 4.6 Continue response

The `Continue` result tells the router to skip automatic HTTP request resolution.  
Use it when you want the handler to take full control of writing the response manually.

```go
w.WriteHeader(http.StatusOK)
w.Header().Set("Content-Type", "application/xml")
w.Write([]byte("Hello World"))

return result.Continue()
```

#### 4.7 Next response

The `Next` is usually used in the request handlers to tell the router that the proccess can continue the execution, it's basically an Ok response without defined payload.

```go
if user.IsAdmin() {
	return result.Next()
}

return result.Err(http.StatusUnauthorized)
```

#### 4.8 Accessors

- `Status()` → Returns the HTTP status code associated with the `Result`.
- `Encoder()` → Returns the `ResultEncoder` used to serialize the payload.
- `Payload()` → Returns the payload of the `Result`.
- `Ok()` → Returns `true` if the result is successful.
- `Err()` → Returns `true` if the result represents an error.

---

### 5. Docs

The `docs` package models routes, payloads, parameters, and responses for generating documentation.

#### 5.1 Viewer

**IDocViewer** provides an interface for implementing documentation viewers that will expose the project routing documentation.

```go
// IDocViewer defines an interface for a documentation viewer.
type IDocViewer interface {
	// Handlers returns a list of handlers that expose the documentation endpoints.
	Handlers() []DocViewerHandler
	// RegisterGroup registers a route group and its associated documentation.
	RegisterGroup(group string, data DocGroup) IDocViewer
	// RegisterRoute registers a single route operation and its documentation.
	RegisterRoute(route DocOperation) IDocViewer
}
```

#### 5.2 Swagger Viewer

The `swagger` package provides an implementation of IDocViewer for OpenAPI 3.0.

```go
route := router.NewRouter()

options := swagger.OpenAPI3ViewerOptions{
    Version:   "v1.0.0",
    EnableTLS: true,
    OnlyTLS:   false,
    Port:      8080,
    PortTLS:   8081,
    FileYML:   "swagger.yaml",
}

viewer := swagger.NewViewer()
viewer.Logger(customLog)
viewer.Load(options)

route.DocViewer(viewer)
```

If a Swagger file path is provided in the options, it will be loaded as the base OpenAPI definition. You can then override or extend it using the given options.

#### 5.3 No-op viewer

If you don’t want documentation, the no-op viewer is used by default:

```go
route := router.NewRouter()
viewer := docs.VoidViewer()
route.DocViewer(viewer)
```

---

#### 5.4 Payloads

```go
payload := docs.DocJsonPayload[Example]("Example JSON response")
payload := docs.DocXmlPayload[Example]("Example XML response")
payload := docs.DocText("Plain text response")
```

#### 5.5 Tags

```go
tags := docs.DocTags("auth", "users")
```

---

### 6. Flags

The configuration is initialized from the `.env` file (or environment variables). The following keys are supported:

| Variable              | Description                         | Default |
|-----------------------|-------------------------------------|---------|
| `GO_WEB_DEV`          | Enables or disables development mode | false   |
| `GO_WEB_TRACE_REQUEST`| Enables or disables HTTP request tracing | false   |

## Example

```go

const PLACE = "place"
const PLACE_DESC = "Place description"

const TEAPOT_DESC = "I'm a teapot"

type DtoPlace struct {
    Place string `json:"place" description:"Place"`
}

type DtoGreetings struct {
    User    string `json:"user" description:"User"`
    Message string `json:"message" description:"Message"`
}

func handler(w http.ResponseWriter, r *http.Request, ctx *router.Context) result.Result {
	user := ctx.Getz("username").
		Stringd("anonymous")
	return result.CustomOk(fmt.Sprintf("hello %v from world", user), &CustomEncoder{})
}

func main() {
    route := router.NewRouter()
    route.Logger(customLog)

    options := swagger.OpenAPI3ViewerOptions{
        Version:   "v1.0.0",
        EnableTLS: true,
        OnlyTLS:   false,
        Port:      8080,
        PortTLS:   8081,
        FileYML:   "swagger.yaml",
    }

    viewer := swagger.NewViewer()
    viewer.Logger(customLog)
    viewer.Load(options)

    route.DocViewer(viewer)

    cors := router.PermissiveCors()
    route.Cors(cors)

    route.Contextualizer(func(http.ResponseWriter, *http.Request) (*router.Context, error) {
        context := router.NewContext()
        context.Put("username", "admin")
        return context, nil
    })

    route.PanicHandler(func(w http.ResponseWriter, r *http.Request, rec any) {
        http.Error(w, "Something went wrong", http.StatusInternalServerError)
    })

    doc := docs.DocRoute{
        Description: "description",
        Parameters: docs.DocParameters{
            PLACE: PLACE_DESC,
        },
        Request: docs.DocJsonPayload[DtoPlace](),
        Responses: docs.DocResponses{
            "200": docs.DocXmlPayload[[]DtoGreetings](),
            "418": docs.DocText(TEAPOT_DESC),
        },
    }

    route.RouteDocument("GET", handler, "/hello/{%s}", doc)
    
    route.Listen(":8080", router.WithReadHeaderTimeout(10*time.Second))
}
```

---

## Migrating from v0.11

- `Listen`, `ListenTLS` and `ListenWithTLS` take addresses (`":8080"`, `"127.0.0.1:8080"`) instead of port numbers, plus optional `ServerOption`s. A string of digits is still read as a port.
- Routers keep their routes on their own mux: `NewRouterWithMux` and `Handler()` are new; `NewRouter()` still uses `http.DefaultServeMux`.
- Response headers set by encoders (`Content-Type`) are now sent; before they were written after the status line and lost on real connections.
- `PanicHandler` can be registered (it panicked in v0.11), and a request that reaches the router without a handler gets `404`.
- `ListenWithTLS` redirects with `308` instead of `301`, and builds the target from the request host and the TLS port.
- CORS follows the rules in section 2: only the matching origin is echoed, unlisted origins and requests without `Origin` get no CORS headers, preflights are answered with `204`, and `PermissiveCors()` no longer allows credentials.
- `ListenTLS` and `ListenWithTLS` take a `TLSStrategy` (`TLSFromFiles`, `TLSFromConfig`) instead of the certificate and key paths.
- `Context`, `ShutdownTimeout` and `OnListen` are new; `Listen` returns `nil` when the context stops the server, and `ListenWithTLS` stops both servers when either one stops.
- `JsonErr`, `XmlErr` and `CustomErr` are now errors: they stop group contextualizers and reach the `ErrorHandler`.
- Plain-text responses carry `charset=utf-8`, and text errors no longer end with a newline.
- Requires Go 1.27.

---

## License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.
