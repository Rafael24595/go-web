package swagger

type OpenAPI3 struct {
	OpenAPI      string              `json:"openapi" yaml:"openapi"`
	Info         Info                `json:"info" yaml:"info"`
	Servers      []Server            `json:"servers,omitempty" yaml:"servers,omitempty"`
	Paths        map[string]PathItem `json:"paths" yaml:"paths"`
	Components   Components          `json:"components,omitempty" yaml:"components,omitempty"`
	Tags         []Tag               `json:"tags,omitempty" yaml:"tags,omitempty"`
	ExternalDocs *ExternalDocs       `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}

type Info struct {
	Title          string   `json:"title" yaml:"title"`
	Description    string   `json:"description,omitempty" yaml:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty" yaml:"contact,omitempty"`
	License        *License `json:"license,omitempty" yaml:"license,omitempty"`
	Version        string   `json:"version" yaml:"version"`
}

type Contact struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	URL   string `json:"url,omitempty" yaml:"url,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

type License struct {
	Name string `json:"name" yaml:"name"`
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`
}

type Server struct {
	URL         string                    `json:"url" yaml:"url"`
	Description string                    `json:"description,omitempty" yaml:"description,omitempty"`
	Variables   map[string]ServerVariable `json:"variables,omitempty" yaml:"variables,omitempty"`
}

type ServerVariable struct {
	Enum        []string `json:"enum,omitempty" yaml:"enum,omitempty"`
	Default     string   `json:"default" yaml:"default"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
}

type PathItem struct {
	Summary     string      `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Get         *Operation  `json:"get,omitempty" yaml:"get,omitempty"`
	Put         *Operation  `json:"put,omitempty" yaml:"put,omitempty"`
	Post        *Operation  `json:"post,omitempty" yaml:"post,omitempty"`
	Delete      *Operation  `json:"delete,omitempty" yaml:"delete,omitempty"`
	Options     *Operation  `json:"options,omitempty" yaml:"options,omitempty"`
	Head        *Operation  `json:"head,omitempty" yaml:"head,omitempty"`
	Patch       *Operation  `json:"patch,omitempty" yaml:"patch,omitempty"`
	Trace       *Operation  `json:"trace,omitempty" yaml:"trace,omitempty"`
	Servers     []Server    `json:"servers,omitempty" yaml:"servers,omitempty"`
	Parameters  []Parameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type Operation struct {
	Tags        []string              `json:"tags,omitempty" yaml:"tags,omitempty"`
	Summary     string                `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses" yaml:"responses"`
	Deprecated  bool                  `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Security    []map[string][]string `json:"security,omitempty" yaml:"security,omitempty"`
	Servers     []Server              `json:"servers,omitempty" yaml:"servers,omitempty"`
}

type Parameter struct {
	Ref         string  `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Name        string  `json:"name" yaml:"name"`
	In          string  `json:"in" yaml:"in"`
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool    `json:"required,omitempty" yaml:"required,omitempty"`
	Deprecated  bool    `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	AllowEmpty  bool    `json:"allowEmptyValue,omitempty" yaml:"allowEmptyValue,omitempty"`
	Schema      *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type RequestBody struct {
	Ref         string               `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Description string               `json:"description,omitempty" yaml:"description,omitempty"`
	Content     map[string]MediaType `json:"content" yaml:"content"`
	Required    bool                 `json:"required,omitempty" yaml:"required,omitempty"`
}

type MediaType struct {
	Schema   *Schema            `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example  interface{}        `json:"example,omitempty" yaml:"example,omitempty"`
	Examples map[string]Example `json:"examples,omitempty" yaml:"examples,omitempty"`
}

type Example struct {
	Ref           string      `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Summary       string      `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description   string      `json:"description,omitempty" yaml:"description,omitempty"`
	Value         interface{} `json:"value,omitempty" yaml:"value,omitempty"`
	ExternalValue string      `json:"externalValue,omitempty" yaml:"externalValue,omitempty"`
}

type Response struct {
	Ref         string               `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Description string               `json:"description" yaml:"description"`
	Headers     map[string]Header    `json:"headers,omitempty" yaml:"headers,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
	Links       map[string]Link      `json:"links,omitempty" yaml:"links,omitempty"`
}

type Header struct {
	Ref         string  `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type Link struct {
	Ref         string                 `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	OperationID string                 `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type Components struct {
	Schemas         map[string]Schema         `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	Responses       map[string]Response       `json:"responses,omitempty" yaml:"responses,omitempty"`
	Parameters      map[string]Parameter      `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Examples        map[string]Example        `json:"examples,omitempty" yaml:"examples,omitempty"`
	RequestBodies   map[string]Parameter      `json:"requestBodies,omitempty" yaml:"requestBodies,omitempty"`
	Headers         map[string]Header         `json:"headers,omitempty" yaml:"headers,omitempty"`
	Links           map[string]Link           `json:"links,omitempty" yaml:"links,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
}

type Schema struct {
	Ref                  string             `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Title                string             `json:"title,omitempty" yaml:"title,omitempty"`
	Type                 string             `json:"type,omitempty" yaml:"type,omitempty"`
	Format               string             `json:"format,omitempty" yaml:"format,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items                *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
	Required             []string           `json:"required,omitempty" yaml:"required,omitempty"`
	Enum                 []interface{}      `json:"enum,omitempty" yaml:"enum,omitempty"`
	Description          string             `json:"description,omitempty" yaml:"description,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	XML                  *XML               `json:"xml,omitempty" yaml:"xml,omitempty"`
}

func NewSchema() *Schema {
	return &Schema{
		Ref:                  "",
		Title:                "",
		Type:                 "object",
		Format:               "",
		Properties:           make(map[string]*Schema),
		Items:                nil,
		Required:             make([]string, 0),
		Enum:                 nil,
		Description:          "",
		AdditionalProperties: nil,
	}
}

type SecurityScheme struct {
	Ref          string `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type         string `json:"type" yaml:"type"`
	Description  string `json:"description,omitempty" yaml:"description,omitempty"`
	Name         string `json:"name,omitempty" yaml:"name,omitempty"`
	In           string `json:"in,omitempty" yaml:"in,omitempty"`
	Scheme       string `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty" yaml:"bearerFormat,omitempty"`
}

type Tag struct {
	Name         string        `json:"name" yaml:"name"`
	Description  string        `json:"description,omitempty" yaml:"description,omitempty"`
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}

type ExternalDocs struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	URL         string `json:"url" yaml:"url"`
}

type XML struct {
	Name      string `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Prefix    string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	Attribute bool   `json:"attribute,omitempty" yaml:"attribute,omitempty"`
	Wrapped   bool   `json:"wrapped,omitempty" yaml:"wrapped,omitempty"`
}
