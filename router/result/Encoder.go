package result

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"reflect"
)

// ResultEncoder defines the interface for serializing a Result payload
// into a specific format (JSON, XML, plain text, etc.) and providing
// the appropriate HTTP headers.
type ResultEncoder interface {
	Encode(payload any) ([]byte, error)
	Headers() map[string]string
}

type jsonEncoder struct{}

// NewJsonEncoder creates a new JSON encoder.
func NewJsonEncoder() ResultEncoder {
	return &jsonEncoder{}
}

// Encode serializes the payload into indented JSON.
// Returns an error if the payload cannot be marshalled.
func (e *jsonEncoder) Encode(payload any) ([]byte, error) {
	if payload == nil {
		return make([]byte, 0), nil
	}

	payloadJson, err := json.MarshalIndent(payload, "", " ")
	if err != nil {
		message := fmt.Sprintf("Error marshalling entity to JSON: %s", err.Error())
		return make([]byte, 0), errors.New(message)
	}
	return payloadJson, nil
}

// Headers returns the HTTP Content-Type header for JSON responses.
func (e *jsonEncoder) Headers() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}

type xmlEncoder struct{}

// NewXmlEncoder creates a new XML encoder.
func NewXmlEncoder() ResultEncoder {
	return &xmlEncoder{}
}

// Encode serializes the payload into indented XML.
// Returns an error if the payload cannot be marshalled.
func (e *xmlEncoder) Encode(payload any) ([]byte, error) {
	if payload == nil {
		return make([]byte, 0), nil
	}

	payloadXML, err := xml.MarshalIndent(payload, "", "  ")
	if err != nil {
		message := fmt.Sprintf("Error marshalling entity to XML: %s", err.Error())
		return make([]byte, 0), errors.New(message)
	}

	return payloadXML, nil
}

// Headers returns the HTTP Content-Type header for XML responses.
func (e *xmlEncoder) Headers() map[string]string {
	return map[string]string{
		"Content-Type": "application/xml",
	}
}

type textEncoder struct{}

// NewTextEncoder creates a new plain-text encoder.
func NewTextEncoder() ResultEncoder {
	return &textEncoder{}
}

// Encode converts the payload into its string representation.
func (e *textEncoder) Encode(payload any) ([]byte, error) {
	if payload == nil {
		return make([]byte, 0), nil
	}

	t := reflect.TypeOf(payload)
	switch t.Kind() {
	case reflect.String:
		return []byte(payload.(string)), nil
	case reflect.Ptr, reflect.Struct, reflect.Map, reflect.Array, reflect.Slice:
		return NewJsonEncoder().Encode(payload)
	default:
		payloadStr := fmt.Sprintf("%v", payload)
		return []byte(payloadStr), nil
	}
}

// Headers returns the HTTP Content-Type header for plain-text responses.
func (e *textEncoder) Headers() map[string]string {
	return map[string]string{
		"Content-Type": "text/plain",
	}
}
