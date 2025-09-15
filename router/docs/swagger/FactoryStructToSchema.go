package swagger

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"strings"

	"github.com/Rafael24595/go-web/router/docs"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type seen struct {
	ref    string
	name   string
	schema Schema
}

// FactoryStructToSchema builds OpenAPI 3.0 schema definitions from Go structs.
// It uses reflection to walk struct fields, inspect tags, and generate JSON/XML schemas.
type FactoryStructToSchema struct {
	seen map[reflect.Type]map[docs.MediaType]seen
}

// NewFactoryStructToSchema creates a new factory instance.
func NewFactoryStructToSchema() *FactoryStructToSchema {
	return &FactoryStructToSchema{
		seen: make(map[reflect.Type]map[docs.MediaType]seen),
	}
}

// Components returns all schemas collected so far as OpenAPI components.
func (f *FactoryStructToSchema) Components() *Components {
	schemas := make(map[string]Schema)
	for _, s := range f.seen {
		for _, r := range s {
			schemas[r.name] = r.schema
		}
	}
	return &Components{
		Schemas: schemas,
	}
}

// MakeSchema creates a schema reference for a given payload.
// - If the payload is a struct, it will be added to the components.
// - If the payload is a slice/array, it wraps the item schema in an "array" type.
func (f *FactoryStructToSchema) MakeSchema(media docs.MediaType, root any) (*Schema, error) {
	t := reflect.TypeOf(root)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	ref, isVector, err := f.collectSchema(media, t)
	if err != nil {
		return nil, err
	}

	if isVector {
		return &Schema{
			Items: &Schema{
				Ref: ref,
			},
		}, nil
	}

	return &Schema{
		Ref: ref,
	}, nil
}

func (f *FactoryStructToSchema) collectSchema(media docs.MediaType, t reflect.Type) (string, bool, error) {
	isVector := f.isVector(t)
	t = f.deferencePointer(t)

	if t.Kind() != reflect.Struct {
		return "", isVector, nil
	}

	if refs, ok := f.seen[t]; ok {
		if ref, ok := refs[media]; ok {
			return ref.ref, isVector, nil
		}
	}

	name, mediaName := f.makeStructName(media, t)
	ref := f.makeRefString(mediaName)

	f.putSeen(t, media, seen{
		ref:  ref,
		name: mediaName,
	})

	schema, err := f.makeSchema(media, t)
	if err != nil {
		return "", isVector, err
	}

	if schema != nil && media == docs.XML {
		schema.XML = &XML{
			Name: name,
			Wrapped: true,
		}
	}

	f.seen[t][media] = seen{
		ref:    ref,
		name:   mediaName,
		schema: *schema,
	}

	f.putSeen(t, media, seen{
		ref:    ref,
		name:   mediaName,
		schema: *schema,
	})

	return ref, isVector, nil
}

func (f *FactoryStructToSchema) putSeen(t reflect.Type, media docs.MediaType, schema seen) {
	if _, ok := f.seen[t]; !ok {
		f.seen[t] = make(map[docs.MediaType]seen)
	}

	f.seen[t][media] = schema
}

func (f *FactoryStructToSchema) makeSchema(media docs.MediaType, t reflect.Type) (*Schema, error) {
	t = f.deferencePointer(t)

	if t.Kind() != reflect.Struct {
		return nil, nil
	}

	schema := NewSchema()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.Anonymous || f.isMiscField(field) {
			continue
		}

		name := field.Name
		isRequired := f.canBeRequired(field)
		ref, err := f.inferSchema(media, field.Type)
		if err != nil {
			return nil, err
		}

		ref.Description = field.Tag.Get("description")

		switch media {
		case docs.XML:
			if xmlTag, xmlOmitempty, xmlRef := f.isXmlField(field, ref); xmlRef != nil {
				name = xmlTag
				isRequired = isRequired && !xmlOmitempty
				ref = xmlRef
			}
		case docs.JSON:
			if jsonTag, jsonOmitempty, jsonRef := f.isJsonField(field, ref); jsonRef != nil {
				name = jsonTag
				isRequired = isRequired && !jsonOmitempty
				ref = jsonRef
			}
		}

		schema = f.addProperty(schema, name, ref, isRequired)
	}

	return schema, nil
}

func (f *FactoryStructToSchema) isMiscField(field reflect.StructField) bool {
	return field.Type == reflect.TypeOf(xml.Name{})
}

func (f *FactoryStructToSchema) addProperty(schema *Schema, name string, property *Schema, isRequired bool) *Schema {
	schema.Properties[name] = property
	if isRequired {
		schema.Required = append(schema.Required, name)
	}
	return schema
}

func (f *FactoryStructToSchema) isJsonField(field reflect.StructField, ref *Schema) (string, bool, *Schema) {
	attribute := field.Tag.Get("json")

	tag := strings.Split(attribute, ",")[0]
	if tag == "" || tag == "-" {
		return "", false, nil
	}

	omitEmpty := strings.Contains(attribute, "omitempty")

	return tag, omitEmpty, ref
}

func (f *FactoryStructToSchema) isXmlField(field reflect.StructField, ref *Schema) (string, bool, *Schema) {
	attribute := field.Tag.Get("xml")

	tag := strings.Split(attribute, ",")[0]
	if tag == "" || tag == "-" {
		return "", false, nil
	}

	wrapper := ""
	if fragments := strings.Split(tag, ">"); len(fragments) > 1 {
		tag = fragments[0]
		wrapper = fragments[1]
	}

	if wrapper != "" {
		ref = &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				wrapper: ref,
			},
		}
	}

	attr := strings.Contains(attribute, "attr")
	ref.XML = &XML{
		Name:      tag,
		Attribute: attr,
	}

	omitEmpty := strings.Contains(attribute, "omitempty")
	return tag, omitEmpty, ref
}

func (f *FactoryStructToSchema) canBeRequired(field reflect.StructField) bool {
	return field.Type.Kind() != reflect.Ptr &&
		field.Type.Kind() != reflect.Slice &&
		field.Type.Kind() != reflect.Map
}

func (f *FactoryStructToSchema) inferSchema(media docs.MediaType, fieldType reflect.Type) (*Schema, error) {
	switch fieldType.Kind() {
	case reflect.Ptr:
		return f.inferSchema(media, fieldType.Elem())
	case reflect.Struct:
		return f.inferStruct(media, fieldType)
	case reflect.Slice, reflect.Array:
		return f.inferArray(media, fieldType)
	case reflect.Map:
		return f.inferMap(media, fieldType)
	case reflect.String:
		return &Schema{Type: "string"}, nil
	case reflect.Bool:
		return &Schema{Type: "boolean"}, nil
	case reflect.Int, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: "integer"}, nil
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number"}, nil
	default:
		return &Schema{Type: "string"}, nil
	}
}

func (f *FactoryStructToSchema) inferStruct(media docs.MediaType, fieldType reflect.Type) (*Schema, error) {
	ref, isVector, err := f.collectSchema(media, fieldType)
	if err != nil {
		return nil, err
	}

	if isVector {
		return &Schema{
			Items: &Schema{
				Ref: ref,
			},
		}, nil
	}

	return &Schema{Ref: ref}, nil
}

func (f *FactoryStructToSchema) inferArray(media docs.MediaType, fieldType reflect.Type) (*Schema, error) {
	itemRef, err := f.inferSchema(media, fieldType.Elem())
	if err != nil {
		return nil, err
	}

	return &Schema{
		Type:  "array",
		Items: itemRef,
	}, nil
}

func (f *FactoryStructToSchema) inferMap(media docs.MediaType, fieldType reflect.Type) (*Schema, error) {
	properties, err := f.inferSchema(media, fieldType.Elem())
	if err != nil {
		return nil, err
	}

	return &Schema{
		Type:                 "object",
		AdditionalProperties: properties,
	}, nil
}

func (f *FactoryStructToSchema) deferencePointer(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		t = t.Elem()
	}
	return t
}

func (f *FactoryStructToSchema) isVector(t reflect.Type) bool {
	return t.Kind() == reflect.Slice || t.Kind() == reflect.Array
}

func (f *FactoryStructToSchema) makeStructName(media docs.MediaType, t reflect.Type) (string, string) {
	name := t.Name()
	if name == "" {
		name = "Anon"
	}

	mediaName := f.makeMediaName(media, t.PkgPath(), name)

	if xmlName, ok := f.hasXmlRoot(t); ok {
		return xmlName, mediaName
	}

	return name, mediaName
}

func (f *FactoryStructToSchema) hasXmlRoot(t reflect.Type) (string, bool) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.Type == reflect.TypeOf(xml.Name{}) {
			attribute := field.Tag.Get("xml")
			tag := strings.Split(attribute, ",")[0]
			return tag, tag != "" && tag != "-"
		}
	}

	return "", false
}

func (f *FactoryStructToSchema) makeMediaName(media docs.MediaType, pkg, name string) string {
	switch media {
	case docs.XML:
		media = "xml"
	case docs.JSON:
		media = "json"
	default:
		media = ""
	}

	fragments := strings.Split(pkg, "/")
	pkgFormat := fragments[len(fragments)-1]

	mediaFormat := string(media)
	nameFormat := name
	if media != "" {
		caser := cases.Title(language.Und, cases.NoLower)
		mediaFormat = caser.String(mediaFormat)
		pkgFormat = caser.String(pkgFormat)
		nameFormat = caser.String(nameFormat)
	}

	return fmt.Sprintf("%s_%s_%s", mediaFormat, pkgFormat, nameFormat)
}

func (f *FactoryStructToSchema) makeRefString(name string) string {
	return fmt.Sprintf("#/components/schemas/%s", name)
}
