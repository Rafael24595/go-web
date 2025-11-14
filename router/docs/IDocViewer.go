package docs

import (
	"net/http"
	"strings"
)

type handler = func(http.ResponseWriter, *http.Request)

// DocResponses maps HTTP status codes or response identifiers to DocPayloads.
type DocResponses map[StatusCode]DocPayload
// DocParameters maps parameter names to their description.
type DocParameters map[string]string

// ParameterType defines the location type of a request parameter.
type ParameterType string

const (
	QUERY ParameterType = "query"
	PATH  ParameterType = "path"
)

// MediaType defines the MIME type of a request or response payload.
type MediaType string

const (
	JSON MediaType = "application/json"
	XML  MediaType = "application/xml"
)

// IDocViewer defines an interface for a documentation viewer.
type IDocViewer interface {
	// Handlers returns a list of handlers that expose the documentation endpoints.
	Handlers() []DocViewerHandler
	// RegisterGroup registers a route group and its associated documentation.
	RegisterGroup(group string, data DocGroup) IDocViewer
	// RegisterRoute registers a single route operation and its documentation.
	RegisterRoute(route DocOperation) IDocViewer
}

// DocViewerSources represents a documented source route.
type DocViewerSources struct {
	Name        string `json:"name"`
	Route       string `json:"route"`
	Description string `json:"description"`
}

// DocViewerHandler describes a single documentation handler.
type DocViewerHandler struct {
	Method      string
	Route       string
	Handler     handler
	Name        string
	Description string
}

// DocGroup represents a group of routes sharing headers, cookies, or response types.
type DocGroup struct {
	Headers   DocParameters
	Cookies   DocParameters
	Responses DocResponses
}

// DocRoute represents the documentation for a single route.
type DocRoute struct {
	Description string
	Parameters  DocParameters
	Query       DocParameters
	Files       DocParameters
	Cookies     DocParameters
	Request     DocPayload
	Responses   DocResponses
	Tags        *[]string
}

// DocOperation represents a documented API operation, combining route info and documentation.
type DocOperation struct {
	Description string
	Method      string
	BasePath    string
	Path        string
	Parameters  DocParameters
	Query       DocParameters
	Files       DocParameters
	Cookies     DocParameters
	Request     DocPayload
	Responses   DocResponses
	Tags        *[]string
}

// DocPayload represents a request or response body and its metadata.
type DocPayload struct {
	Payload     any
	MediaType   MediaType
	Description string
}

// DocXmlPayload creates a DocPayload with XML media type.
func DocXmlPayload[T any](description ...string) DocPayload {
	var xml T
	return docPayload(xml, XML, description...)
}

// DocJsonPayload creates a DocPayload with JSON media type.
func DocJsonPayload[T any](description ...string) DocPayload {
	var json T
	return docPayload(json, JSON, description...)
}

// DocText creates a DocPayload representing text or empty JSON body.
func DocText(description ...string) DocPayload {
	return docPayload("", JSON, description...)
}

func docPayload(item any, media MediaType, description ...string) DocPayload {
	return DocPayload{
		Payload:     item,
		MediaType:   media,
		Description: strings.Join(description, ""),
	}
}

// DocTags creates a pointer to a slice of tags for route grouping.
func DocTags(tags ...string) *[]string {
	return &tags
}
