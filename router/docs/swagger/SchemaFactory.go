package swagger

import (
	"encoding"
	"fmt"
	"reflect"

	"github.com/Rafael24595/go-web/router/docs"
)

var textMarshalerType = reflect.TypeFor[encoding.TextMarshaler]()

type component struct {
	ref    string
	name   string
	schema Schema
}

// schemaFactory builds OpenAPI 3.0 schema definitions from Go structs.
// It uses reflection to walk struct fields, inspect tags, and generate JSON/XML schemas.
type schemaFactory struct {
	names     map[string]reflect.Type
	component map[reflect.Type]map[docs.MediaType]component
	defined   map[reflect.Type]Schema
}

// newSchemaFactory creates a new factory instance.
func newSchemaFactory() *schemaFactory {
	return &schemaFactory{
		names:     make(map[string]reflect.Type),
		component: make(map[reflect.Type]map[docs.MediaType]component),
		defined:   make(map[reflect.Type]Schema),
	}
}

// defineType documents every value of type t with schema instead of inferring
// it; it replaces any earlier definition of t.
func (f *schemaFactory) defineType(t reflect.Type, schema Schema) *schemaFactory {
	f.defined[t] = schema
	return f
}

// components returns all schemas collected so far as OpenAPI components.
func (f *schemaFactory) components() *Components {
	schemas := make(map[string]Schema)
	for _, s := range f.component {
		for _, r := range s {
			schemas[r.name] = r.schema
		}
	}
	return &Components{
		Schemas: schemas,
	}
}

// makeSchema describes a payload as the media type's encoder writes it.
// Structs are added to the components and referenced; any other type is
// described inline (an array of its items, a map, a basic type). A nil
// payload is described as any value.
func (f *schemaFactory) makeSchema(media docs.MediaType, root any) (*Schema, error) {
	meta, err := formatOf(media)
	if err != nil {
		return nil, err
	}

	t := reflect.TypeOf(root)
	if t == nil {
		return &Schema{}, nil
	}

	return f.inferSchema(meta, t)
}

func (f *schemaFactory) collectSchema(format mediaFormat, t reflect.Type) (string, error) {
	if ref, ok := f.lookup(t, format); ok {
		return ref, nil
	}

	info := format.structExtractor(t)

	name := f.uniqueName(info.component, t)
	ref := f.componentRef(name)

	f.collect(t, format.mediaType, component{
		ref:  ref,
		name: name,
	})

	schema, err := f.makeStructSchema(format, t)
	if err != nil {
		return "", err
	}

	schema.XML = info.xml

	f.collect(t, format.mediaType, component{
		ref:    ref,
		name:   name,
		schema: *schema,
	})

	return ref, nil
}

func (f *schemaFactory) lookup(t reflect.Type, format mediaFormat) (string, bool) {
	if c, ok := f.component[t][format.mediaType]; ok {
		return c.ref, true
	}
	return "", false
}

func (f *schemaFactory) collect(t reflect.Type, media docs.MediaType, schema component) {
	if _, ok := f.component[t]; !ok {
		f.component[t] = make(map[docs.MediaType]component)
	}

	f.component[t][media] = schema
}

func (f *schemaFactory) uniqueName(name string, t reflect.Type) string {
	unique := name

	for i := 2; ; i++ {
		owner, taken := f.names[unique]
		if !taken || owner == t {
			f.names[unique] = t
			return unique
		}

		unique = fmt.Sprintf("%s_%d", name, i)
	}
}

func (f *schemaFactory) makeStructSchema(format mediaFormat, t reflect.Type) (*Schema, error) {
	schema := NewSchema()

	for field := range t.Fields() {
		if field.Anonymous || !field.IsExported() {
			continue
		}

		result := format.fieldExtractor(field)
		if result.ignore {
			continue
		}

		ref, err := f.inferSchema(format, field.Type)
		if err != nil {
			return nil, err
		}

		ref.Description = field.Tag.Get("description")

		ref = result.decorate(ref)

		isRequired := f.canBeRequired(field) && !result.omitempty
		schema = f.addProperty(schema, result.name, ref, isRequired)
	}

	return schema, nil
}

func (f *schemaFactory) addProperty(schema *Schema, name string, property *Schema, isRequired bool) *Schema {
	schema.Properties[name] = property
	if isRequired {
		schema.Required = append(schema.Required, name)
	}
	return schema
}

func (f *schemaFactory) canBeRequired(field reflect.StructField) bool {
	return field.Type.Kind() != reflect.Pointer &&
		field.Type.Kind() != reflect.Slice &&
		field.Type.Kind() != reflect.Map
}

func (f *schemaFactory) inferSchema(format mediaFormat, fieldType reflect.Type) (*Schema, error) {
	for fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}

	if schema, ok := f.defined[fieldType]; ok {
		return &schema, nil
	}

	switch {
	case implements(fieldType, format.marshaler):
		return &Schema{}, nil
	case implements(fieldType, textMarshalerType):
		return &Schema{Type: "string"}, nil
	case fieldType.Kind() == reflect.Slice && fieldType.Elem().Kind() == reflect.Uint8:
		bytes := format.bytes
		return &bytes, nil
	}

	switch fieldType.Kind() {
	case reflect.Interface:
		return &Schema{}, nil
	case reflect.Struct:
		return f.inferStruct(format, fieldType)
	case reflect.Slice, reflect.Array:
		return f.inferArray(format, fieldType)
	case reflect.Map:
		return f.inferMap(format, fieldType)
	case reflect.String:
		return &Schema{Type: "string"}, nil
	case reflect.Bool:
		return &Schema{Type: "boolean"}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: "integer"}, nil
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number"}, nil
	default:
		return &Schema{Type: "string"}, nil
	}
}

func (f *schemaFactory) inferStruct(format mediaFormat, fieldType reflect.Type) (*Schema, error) {
	ref, err := f.collectSchema(format, fieldType)
	if err != nil {
		return nil, err
	}

	return &Schema{Ref: ref}, nil
}

func (f *schemaFactory) inferArray(format mediaFormat, fieldType reflect.Type) (*Schema, error) {
	itemRef, err := f.inferSchema(format, fieldType.Elem())
	if err != nil {
		return nil, err
	}

	return &Schema{
		Type:  "array",
		Items: itemRef,
	}, nil
}

func (f *schemaFactory) inferMap(format mediaFormat, fieldType reflect.Type) (*Schema, error) {
	properties, err := f.inferSchema(format, fieldType.Elem())
	if err != nil {
		return nil, err
	}

	return &Schema{
		Type:                 "object",
		AdditionalProperties: properties,
	}, nil
}

func (f *schemaFactory) componentRef(name string) string {
	return fmt.Sprintf("#/components/schemas/%s", name)
}

func implements(t, iface reflect.Type) bool {
	return t.Implements(iface) || reflect.PointerTo(t).Implements(iface)
}
