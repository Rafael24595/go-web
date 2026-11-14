package swagger

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"sync"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/Rafael24595/go-log/log"
	"github.com/Rafael24595/go-web/router/docs"
	"github.com/Rafael24595/go-web/router/internal/paths"
)

const SWAGGER string = "SWAGGER"

const SWAGGER_ROUTE = "/swagger/"
const SWAGGER_JSON = "/swagger/doc.json"

// OpenAPI3Viewer implements the docs.IDocViewer interface
// and exposes API documentation in OpenAPI 3.0 format.
type OpenAPI3Viewer struct {
	build      sync.Once
	logger     log.Log
	data       OpenAPI3
	factory    *schemaFactory
	headers    map[string]map[string]string
	cookies    map[string]map[string]string
	responses  map[string]map[string]Response
	stringData string
}

// NewViewer creates a new OpenAPI3Viewer with default values.
func NewViewer() *OpenAPI3Viewer {
	return &OpenAPI3Viewer{
		data:       OpenAPI3{},
		logger:     log.Default(),
		factory:    newSchemaFactory(),
		headers:    make(map[string]map[string]string),
		cookies:    make(map[string]map[string]string),
		responses:  make(map[string]map[string]Response),
		stringData: "",
	}
}

// Logger sets the logger for the viewer and returns itself.
func (v *OpenAPI3Viewer) Logger(logger log.Log) *OpenAPI3Viewer {
	v.logger = logger
	return v
}

// Load sets the definition the viewer starts from: OpenAPI 3.0.0, the fields
// set by options (applied in order) and this server ("/") as the only server.
// When an option fails, Load returns its error and leaves the viewer unchanged.
func (v *OpenAPI3Viewer) Load(options ...OpenAPI3ViewerOption) error {
	data, err := applyOptions(options...)
	if err != nil {
		return err
	}

	data.Servers = []Server{
		{URL: "/", Description: "This server"},
	}

	v.logger.Customf(SWAGGER, "Swagger interface displayed on %s", SWAGGER_ROUTE)
	v.logger.Customf(SWAGGER, "Swagger JSON displayed on %s", SWAGGER_JSON)

	v.data = *data

	return nil
}

// DefineType documents every value of type t with schema instead of inferring
// it from its fields: for types that encode themselves (uuid.UUID, a decimal)
// or whose encoding has a format OpenAPI names.
// Call it before the routes are registered.
//
// Returns the viewer itself for fluent configuration.
func (v *OpenAPI3Viewer) DefineType(t reflect.Type, schema Schema) *OpenAPI3Viewer {
	v.factory.defineType(t, schema)
	return v
}

// RegisterGroup registers shared documentation (headers, cookies, responses)
// for a group of routes identified by a prefix.
func (v *OpenAPI3Viewer) RegisterGroup(group string, data docs.DocGroup) docs.IDocViewer {
	v.groupHeaders(group, data.Headers)
	v.groupCookies(group, data.Cookies)
	v.groupResponses(group, data.Responses)
	return v
}

func (v *OpenAPI3Viewer) groupHeaders(group string, headers map[string]string) docs.IDocViewer {
	item, ok := v.headers[group]
	if !ok {
		item = make(map[string]string)
	}

	maps.Copy(item, headers)

	v.headers[group] = item

	return v
}

func (v *OpenAPI3Viewer) groupCookies(group string, cookies map[string]string) docs.IDocViewer {
	item, ok := v.cookies[group]
	if !ok {
		item = make(map[string]string)
	}

	maps.Copy(item, cookies)

	v.cookies[group] = item

	return v
}

func (v *OpenAPI3Viewer) groupResponses(group string, responses map[docs.StatusCode]docs.DocPayload) docs.IDocViewer {
	item, ok := v.responses[group]
	if !ok {
		item = make(map[string]Response)
	}

	result := v.makeResponsesFromMap(responses)

	maps.Copy(item, result)

	v.responses[group] = item

	return v
}

// Handlers returns the HTTP handlers for the Swagger UI and JSON definition.
//
// Routes:
//   - GET /swagger/         → Swagger UI
//   - GET /swagger/doc.json → OpenAPI 3 JSON document
func (v *OpenAPI3Viewer) Handlers() []docs.DocViewerHandler {
	return []docs.DocViewerHandler{
		{
			Method:      http.MethodGet,
			Route:       SWAGGER_ROUTE,
			Handler:     httpSwagger.WrapHandler,
			Name:        "OAS3",
			Description: "OpenAPI 3.0 view",
		},
		{
			Method:      http.MethodGet,
			Route:       SWAGGER_JSON,
			Handler:     v.doc,
			Name:        "OAS3 JSON",
			Description: "OpenAPI 3.0 definition",
		},
	}
}

func (v *OpenAPI3Viewer) doc(w http.ResponseWriter, r *http.Request) {
	v.build.Do(func() {
		v.data.Components = *v.factory.components()

		data, err := json.Marshal(v.data)
		if err != nil {
			v.logger.Error(err)
		}

		v.stringData = string(data)
	})

	if v.stringData == "" {
		http.Error(w, "the OpenAPI definition could not be built", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	_, err := w.Write([]byte(v.stringData))
	if err != nil {
		v.logger.Error(err)
	}
}

// RegisterRoute registers an individual route operation into the OpenAPI 3 definition.
//
// It maps the route’s method, path, parameters, request, and responses into
// the corresponding OpenAPI structures.
func (v *OpenAPI3Viewer) RegisterRoute(route docs.DocOperation) docs.IDocViewer {
	if v.data.Paths == nil {
		v.data.Paths = make(map[string]PathItem)
	}

	path := fmt.Sprintf("%s%s", route.BasePath, route.Path)

	pathItem, ok := v.data.Paths[path]
	if !ok {
		pathItem = PathItem{}
	}

	operation := &Operation{
		Tags:        makeTags(route),
		Description: route.Description,
		Parameters:  v.makeParameters(path, route),
		RequestBody: v.makeRequest(route),
		Responses:   v.makeResponses(path, route),
	}

	switch route.Method {
	case "GET":
		pathItem.Get = operation
	case "POST":
		pathItem.Post = operation
	case "PUT":
		pathItem.Put = operation
	case "DELETE":
		pathItem.Delete = operation
	case "PATCH":
		pathItem.Patch = operation
	case "HEAD":
		pathItem.Head = operation
	case "OPTIONS":
		pathItem.Options = operation
	default:
		v.logger.Warningf("Unsupported HTTP method: %s", route.Method)
	}

	v.logger.Customf(SWAGGER, "Route registered: [%s] %s", route.Method, path)

	v.data.Paths[path] = pathItem
	return v
}

func (v *OpenAPI3Viewer) makeParameters(path string, route docs.DocOperation) []Parameter {
	parameters := make([]Parameter, 0)

	groups := paths.Groups(
		path, slices.Collect(maps.Keys(v.headers)),
	)

	for _, group := range groups {
		headers := v.headers[group]
		names := slices.Sorted(maps.Keys(headers))

		for _, name := range names {
			parameters = append(parameters, Parameter{
				Name:        name,
				In:          "header",
				Description: headers[name],
				Required:    true,
			})
		}
	}

	groups = paths.Groups(
		path, slices.Collect(maps.Keys(v.cookies)),
	)

	for _, group := range groups {
		cookies := v.cookies[group]
		names := slices.Sorted(maps.Keys(cookies))

		for _, name := range names {
			cookie := v.makeCookie(name, cookies[name])
			parameters = append(parameters, cookie)
		}
	}

	if route.Parameters != nil {
		for _, d := range route.Parameters {
			parameters = append(parameters, Parameter{
				Name:        d.Code,
				In:          "path",
				Description: d.Description,
				Required:    true,
			})
		}
	}

	if route.Query != nil {
		for n, d := range route.Query {
			parameters = append(parameters, Parameter{
				Name:        n,
				In:          "query",
				Description: d,
				Required:    false,
			})
		}
	}

	if route.Cookies != nil {
		for n, d := range route.Cookies {
			parameters = append(parameters, v.makeCookie(n, d))
		}
	}

	return parameters
}

func (v *OpenAPI3Viewer) makeCookie(name string, description string) Parameter {
	cookie := Parameter{
		Name:        name,
		In:          "cookie",
		Description: description,
		Required:    false,
	}

	cookie.Schema = &Schema{
		Type: "string",
	}

	return cookie
}

func (v *OpenAPI3Viewer) makeRequest(route docs.DocOperation) *RequestBody {
	content := make(map[string]MediaType)

	if contentType, media := v.makeMainRequest(route); media != nil {
		content[string(contentType)] = *media
	}

	if contentType, media := v.makeFileRequest(route); media != nil {
		content[contentType] = *media
	}

	if len(content) == 0 {
		return nil
	}

	return &RequestBody{
		Description: route.Request.Description,
		Content:     content,
	}
}

func (v *OpenAPI3Viewer) makeMainRequest(route docs.DocOperation) (docs.MediaType, *MediaType) {
	if route.Request.Payload == nil {
		return "", nil
	}

	main, err := v.factory.makeSchema(route.Request.MediaType, route.Request.Payload)
	if err != nil {
		v.logger.Error(err)
		return "", nil
	}

	return route.Request.MediaType, &MediaType{
		Schema: main,
	}
}

func (v *OpenAPI3Viewer) makeFileRequest(route docs.DocOperation) (string, *MediaType) {
	if len(route.Files) == 0 {
		return "", nil
	}

	properties := make(map[string]*Schema)
	for k, d := range route.Files {
		properties[k] = &Schema{
			Type:        "string",
			Format:      "binary",
			Description: d,
		}
	}

	multipart := &Schema{
		Type:       "object",
		Properties: properties,
	}

	return "multipart/form-data", &MediaType{
		Schema: multipart,
	}
}

func (v *OpenAPI3Viewer) makeResponses(path string, route docs.DocOperation) map[string]Response {
	reponses := make(map[string]Response)

	groups := paths.Groups(
		path, slices.Collect(maps.Keys(v.responses)),
	)

	for _, group := range groups {
		maps.Copy(reponses, v.responses[group])
	}

	result := v.makeResponsesFromMap(route.Responses)

	maps.Copy(reponses, result)
	return reponses
}

func (v *OpenAPI3Viewer) makeResponsesFromMap(responses map[docs.StatusCode]docs.DocPayload) map[string]Response {
	if len(responses) == 0 {
		return make(map[string]Response)
	}

	result := make(map[string]Response)
	for status, response := range responses {
		main, err := v.factory.makeSchema(response.MediaType, response.Payload)
		if err != nil {
			v.logger.Error(err)
			return nil
		}

		result[string(status)] = Response{
			Description: response.Description,
			Content: map[string]MediaType{
				string(response.MediaType): {
					Schema: main,
				},
			},
		}
	}

	return result
}

func makeTags(route docs.DocOperation) []string {
	if route.Tags != nil {
		return *route.Tags
	}

	tags := make([]string, 0)
	if route.BasePath != "" {
		tags = append(tags, route.BasePath)
	}

	fragments := strings.Split(route.Path, "/")
	if len(fragments) > 0 && fragments[0] != "" && !strings.HasPrefix(fragments[0], "{") {
		tags = append(tags, fragments[0])
	}

	return tags
}
