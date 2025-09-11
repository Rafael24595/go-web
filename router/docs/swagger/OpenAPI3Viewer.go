package swagger

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/Rafael24595/go-web/router/docs"
	"github.com/Rafael24595/go-web/router/log"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"gopkg.in/yaml.v3"
)

const SWAGGER string = "SWAGGER"

const SWAGGER_ROUTE = "/swagger/"
const SWAGGER_JSON = "/swagger/doc.json"

// OpenAPI3ViewerOptions defines the configuration for the OpenAPI 3.0 viewer.
type OpenAPI3ViewerOptions struct {
	Version   string // API version
	EnableTLS bool   // Whether to expose an HTTPS server URL
	OnlyTLS   bool   // Whether to expose only HTTPS server URL
	Port      int    // HTTP port
	PortTLS   int    // HTTPS port
	FileYML   string // Path to an existing OpenAPI YAML file to preload
}

// OpenAPI3Viewer implements the docs.IDocViewer interface
// and exposes API documentation in OpenAPI 3.0 format.
type OpenAPI3Viewer struct {
	build      sync.Once
	logger     log.Log
	data       OpenAPI3
	factory    *FactoryStructToSchema
	headers    map[string]map[string]string
	cookies    map[string]map[string]string
	responses  map[string]map[string]Response
	stringData string
}

// NewViewer creates a new OpenAPI3Viewer with default values.
func NewViewer() *OpenAPI3Viewer {
	return &OpenAPI3Viewer{
		data:       OpenAPI3{},
		logger:     log.DefaultLogger(),
		factory:    NewFactoryStructToSchema(),
		headers:    make(map[string]map[string]string),
		cookies:    make(map[string]map[string]string),
		responses:  make(map[string]map[string]Response),
		stringData: "",
	}
}

// Logger sets the logger for the viewer and returns itself.
func (v *OpenAPI3Viewer) Logger(logger log.Log) docs.IDocViewer {
	v.logger = logger
	return v
}

// Load loads the OpenAPI 3 specification from a YAML file,
// and configures server URLs based on the provided options.
//
// It automatically registers `http://localhost:{Port}` if OnlyTLS is false,
// and `https://localhost:{PortTLS}` if EnableTLS is true.
func (v *OpenAPI3Viewer) Load(options OpenAPI3ViewerOptions) docs.IDocViewer {
	data, err := loadYAML(options.FileYML)
	if err != nil {
		v.logger.Error(err)
		data = &OpenAPI3{}
	}

	data.Servers = []Server{}

	if !options.OnlyTLS {
		httpURL := fmt.Sprintf("http://localhost:%d", options.Port)
		data.Servers = append(data.Servers, Server{
			URL:         httpURL,
			Description: "HTTP server",
		})
	}

	if options.EnableTLS {
		httpsURL := fmt.Sprintf("https://localhost:%d", options.PortTLS)
		data.Servers = append(data.Servers, Server{
			URL:         httpsURL,
			Description: "HTTPS server",
		})
	}

	data.Info.Version = options.Version

	v.logger.Customf(SWAGGER, "Swagger interface displayed on %s", SWAGGER_ROUTE)
	v.logger.Customf(SWAGGER, "Swagger JSON displayed on %s", SWAGGER_JSON)

	v.data = *data

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

func (v *OpenAPI3Viewer) groupResponses(group string, responses map[string]docs.DocPayload) docs.IDocViewer {
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
		v.data.Components = *v.factory.Components()
		data, err := json.Marshal(v.data)
		if err != nil {
			v.logger.Error(err)
		}
		v.stringData = string(data)
	})

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

	for k, h := range v.headers {
		if strings.HasPrefix(path, k) {
			for n, d := range h {
				parameters = append(parameters, v.makeParameter(n, d, "header"))
			}
		}
	}

	for k, h := range v.cookies {
		if strings.HasPrefix(path, k) {
			for n, d := range h {
				parameters = append(parameters, v.makeCookie(n, d))
			}
		}
	}

	if route.Parameters != nil {
		for n, d := range route.Parameters {
			parameters = append(parameters, v.makeParameter(n, d, "path"))
		}
	}

	if route.Query != nil {
		for n, d := range route.Query {
			parameters = append(parameters, v.makeParameter(n, d, "query"))
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
	cookie := v.makeParameter(name, description, "cookie")
	cookie.Schema = &Schema{
		Type: "string",
	}
	return cookie
}

func (v *OpenAPI3Viewer) makeParameter(name, description, category string) Parameter {
	return Parameter{
		Name:        name,
		In:          category,
		Description: description,
		Required:    true,
	}
}

func (v *OpenAPI3Viewer) makeRequest(route docs.DocOperation) *RequestBody {
	content := make(map[string]MediaType)

	if contentType, media := v.makeMainRequest(route); media != nil {
		content[string(contentType)] = *media
	}

	if contentType, media := v.makeFileRequest(route); media != nil {
		content[contentType] = *media
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

	main, err := v.factory.MakeSchema(route.Request.MediaType, route.Request.Payload)
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
	for k, h := range v.responses {
		if strings.HasPrefix(path, k) {
			maps.Copy(reponses, h)
		}
	}

	result := v.makeResponsesFromMap(route.Responses)

	maps.Copy(result, reponses)

	return result
}

func (v *OpenAPI3Viewer) makeResponsesFromMap(responses map[string]docs.DocPayload) map[string]Response {
	if len(responses) == 0 {
		return make(map[string]Response)
	}

	result := make(map[string]Response)
	for status, response := range responses {
		main, err := v.factory.MakeSchema(response.MediaType, response.Payload)
		if err != nil {
			v.logger.Error(err)
			return nil
		}
		result[status] = Response{
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

func loadYAML(filename string) (*OpenAPI3, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var doc OpenAPI3
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}
