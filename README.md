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

#### Basic route

```go
router.Route("GET", handler, "/hello")
```

#### With parameters

```go
const PLACE = "place"

route.Route("GET", handler, "/hello/{%s}", PLACE)
```

#### With options

```go
options := router.NewHandlerOptions(handler).
    Context(ctxHandler).
    Error(errHandler).
    Panic(panicHandler)

route.RouteWithOptions("GET", options, "/secure")
```

#### With documentation

```go

const PLACE_DESC = "Place description"
const TEAPOT_DESC = "I'm a teapot"

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
```

#### With documentation and options

```go

route.RouteDocumentWithOptions("GET", options, "/secure", doc)
```

#### Context handler

```go
route.Contextualizer(func(http.ResponseWriter, *http.Request) (router.Context, error) {
    context := router.NewContext()
    context.Put("username", "admin")
    return *context, nil
})
```

#### Error handler

```go
route.ErrorHandler(func(http.ResponseWriter, *http.Request, router.Context, result.Result) {
    if err, ok := result.Err(); ok && err != nil {
		http.Error(wrt, err.Error(), result.Status())
		return
	}

	wrt.WriteHeader(result.Status())
})
```

#### Panic handler

```go
route.PanicHandler(func(w http.ResponseWriter, r *http.Request, rec any) {
    message := fmt.Sprintf("Recovered from panic: %v", r)
    http.Error(w, message, http.StatusInternalServerError)
})
```

#### Context

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

---

### 2. CORS

The `Cors` struct provides a fluent API for configuring cross-origin resource sharing.

```go
cors := router.EmptyCors().
    AllowedOrigins("https://example.com").
    AllowedMethods("GET", "POST").
    AllowedHeaders("Authorization").
    AllowCredentials()

route := router.NewRouter().Cors(cors)
```

#### Permissive CORS

Allows all origins, methods, and headers:

```go
cors := router.PermissiveCors()
route := router.NewRouter().Cors(cors)
```

---

### 3 .Input Deserializers

The framework provides built-in helper functions for deserializing request bodies into Go structs. These functions simplify handling JSON and XML payloads inside route handlers.

#### Text Input

Reads the entire request body as raw bytes.
- On success: returns the raw bytes and nil.
- On failure: returns an empty vector of bytes and a *result.Result with status 400 Bad Request.

```go
func handler(w http.ResponseWriter, r *http.Request, ctx router.Context) result.Result {
    bytes, res := router.InputText(r)
    if res != nil {
        return *res
    }

    return result.Ok(string(bytes))
}
```

#### JSON Input

Parses the request body as JSON into a value of type T.
- On success: returns the parsed payload and nil.
- On failure: returns the zero value of T and a *result.Result with status 422 Unprocessable Entity.

```go
type User struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

func handler(w http.ResponseWriter, r *http.Request, ctx router.Context) result.Result {
    user, res := router.InputJson[User](r)
    if res != nil {
        return *res
    }

    return result.JsonOk(map[string]string{
        "message": "Hello " + user.Name,
    })
}
```

#### XML Input

Parses the request body as XML into a value of type T. The decoder supports multiple character sets via golang.org/x/net/html/charset.
- On success: returns the parsed payload and nil.
- On failure: returns the zero value of T and a *result.Result with status 422 Unprocessable Entity.

```go
type Product struct {
    ID    int    `xml:"id"`
    Name  string `xml:"name"`
    Price string `xml:"price"`
}

func handler(w http.ResponseWriter, r *http.Request, ctx router.Context) result.Result {
    product, res := router.InputXml[Product](r)
    if res != nil {
        return *res
    }

    return result.XmlOk(product)
}
```

### 4. Result Handling

The `Result` type standardizes how responses are returned from route handlers.

#### Custom Encoders

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

#### Success Responses (200 OK by default)
- `Ok(payload)` → Successful response (`200 OK`) with plain text encoder.
- `JsonOk(payload)` → Successful response (`200 OK`) encoded as JSON.
- `XmlOk(payload)` → Successful response (`200 OK`) encoded as XML.
- `FileOk(payload)` → Successful response (`200 OK`) with a file encoder.
- `CustomOk(payload, encoder)` → Successful response (`200 OK`) with a custom encoder.

```go
// Plain text
return result.Ok("Operation successful")

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

#### Success Responses with Custom HTTP Status
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

#### Error Responses
- `Err(status, error)` → Error response with a specific HTTP status code and error message, plain text by default.
- `JsonErr(status, payload)` → Error response with a specific HTTP status code, encoded as JSON.
- `XmlErr(status, payload)` → Error response with a specific HTTP status code, encoded as XML.
- `CustomErr(status, payload, encoder)` → Error response with a specific HTTP status code and a custom encoder.

```go
// Plain text error
return result.Err(400, errors.New("Bad request"))

// JSON error
return result.JsonErr(404, map[string]string{"error": "Not found"})

// XML error
return result.XmlErr(500, struct {
    Message string `xml:"message"`
}{"Internal server error"})

// Custom encoder for errors
return result.CustomErr(403, "Forbidden", &CustomEncoder{})
```

#### Responses without payload
- `Accept(status)` → Accept response with no payload (success) and custom HTTP status code.
- `Reject(status)` → Reject response with no payload (failure) and custom HTTP status code.

```go
// Accept with status 202
return result.Accept(202)

// Reject with status 403
return result.Reject(403)
```

#### Continue response

The `Continue` result tells the router to skip automatic HTTP request resolution.  
Use it when you want the handler to take full control of writing the response manually.

```go
w.WriteHeader(http.StatusOK)
w.Header().Set("Content-Type", "application/xml")
w.Write([]byte("Hello World"))

return result.Continue()
```

#### Accessors

- `Status()` → Returns the HTTP status code associated with the `Result`.
- `Encoder()` → Returns the `ResultEncoder` used to serialize the payload.
- `Payload()` → Returns the payload of the `Result`.
- `Ok()` → Returns `true` if the result is successful.
- `Err()` → Returns `true` if the result represents an error.

---

### 5. Docs

The `docs` package models routes, payloads, parameters, and responses for generating documentation.

#### Viewer

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

#### Swagger Viewer

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

#### No-op viewer

If you don’t want documentation, the no-op viewer is used by default:

```go
route := router.NewRouter()
viewer := docs.VoidViewer()
route.DocViewer(viewer)
```

---

#### Payloads

```go
payload := docs.DocJsonPayload[Example]("Example JSON response")
payload := docs.DocXmlPayload[Example]("Example XML response")
payload := docs.DocText("Plain text response")
```

#### Tags

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

func handler(w http.ResponseWriter, r *http.Request, ctx router.Context) result.Result {
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
    route.Cors()

    route.Contextualizer(func(http.ResponseWriter, *http.Request) (router.Context, error) {
        context := router.NewContext()
        context.Put("username", "admin")
        return *context, nil
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
    
    route.Listen(":8080")
}
```

---

## License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.
