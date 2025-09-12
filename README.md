# go-web

A lightweight and extensible Go web framework providing:

- HTTP routing with flexible handlers
- CORS configuration
- Panic and error handling
- Request/response result utilities
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
route.Route("GET", handler, "/hello")
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
	Request: docs.DocJsonPayload(DtoPlace{}),
	Responses: docs.DocResponses{
        "200": docs.DocXmlPayload([]DtoGreetings{}),
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
route.Contextualizer(func(http.ResponseWriter, *http.Request) (Context, error) {
    context := collection.DictionaryEmpty[string, any]()
	context.Put("username", "admin")
	return context, nil
})
```

#### Error handler

```go
route.ErrorHandler(func(http.ResponseWriter, *http.Request, Context, result.Result) {
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

---

### 2. CORS

The `Cors` struct provides a fluent API for configuring cross-origin resource sharing.

```go
cors := router.EmptyCors().
    AllowedOrigins("https://example.com").
    AllowedMethods("GET", "POST").
    AllowedHeaders("Authorization").
    AllowCredentials()
```

#### Permissive CORS

Allows all origins, methods, and headers:

```go
cors := router.PermissiveCors()
route := router.NewRouter().Cors(cors)
```

---

### 3. Result

The `result` package provides a unified way of handling success/error responses.

```go
res := result.Ok(data)              // 200 OK
res := result.Oks(201, data)        // custom status success
res := result.Err(400, err)         // error with status
res := result.Accept(202)           // accepted without payload
res := result.Reject(403)           // reject without payload
```

Accessors:

```go
status := res.Status()
okValue, isOk := res.Ok()
errValue, isErr := res.Err()
```

---

### 4. Docs

The `docs` package models routes, payloads, parameters, and responses for generating documentation.

#### Payloads

```go
payload := docs.DocJsonPayload(Example{}, "Example JSON response")
payload := docs.DocXmlPayload(Example{}, "Example XML response")
payload := docs.DocText("Plain text response")
```

#### Tags

```go
tags := docs.DocTags("auth", "users")
```

#### No-op viewer

If you don’t want documentation:

```go
viewer := docs.VoidViewer()
```

---

### 5. Swagger

The `swagger` package provides struct-to-schema conversion for OpenAPI.

```go
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

---

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

func main() {
    route := router.NewRouter()
    route.Logger(customLog)

    cors := router.PermissiveCors()
    route.Cors(cors)

    route.PanicHandler(func(w http.ResponseWriter, r *http.Request, rec any) {
        http.Error(w, "Something went wrong", http.StatusInternalServerError)
    })

    doc := docs.DocRoute{
        Description: "description",
        Parameters: docs.DocParameters{
            PLACE: PLACE_DESC,
        },
        Request: docs.DocJsonPayload(DtoPlace{}),
        Responses: docs.DocResponses{
            "200": docs.DocXmlPayload([]DtoGreetings{}),
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
