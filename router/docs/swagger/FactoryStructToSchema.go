package swagger

import (
	"fmt"
	"reflect"

	"github.com/Rafael24595/go-web/router/docs"
)

type seen struct {
	ref    string
	name   string
	schema Schema
}

// FactoryStructToSchema builds OpenAPI 3.0 schema definitions from Go structs.
// It uses reflection to walk struct fields, inspect tags, and generate JSON/XML schemas.
type FactoryStructToSchema struct {
	seen  map[reflect.Type]map[docs.MediaType]seen
	types map[reflect.Type]Schema
}

// NewFactoryStructToSchema creates a new factory instance.
func NewFactoryStructToSchema() *FactoryStructToSchema {
	return &FactoryStructToSchema{
		seen:  make(map[reflect.Type]map[docs.MediaType]seen),
		types: make(map[reflect.Type]Schema),
	}
}

// DefineType documents every value of type t with schema instead of inferring
// it; it replaces any earlier definition of t, the built-in ones included.
func (f *FactoryStructToSchema) DefineType(t reflect.Type, schema Schema) *FactoryStructToSchema {
	f.types[t] = schema
	return f
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
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	meta, err := mediaMetaFromType(media)
	if err != nil {
		return nil, err
	}

	return f.inferStruct(meta, t)
}

func (f *FactoryStructToSchema) collectSchema(meta mediaMeta, t reflect.Type) (string, bool, error) {
	isVector := f.isVector(t)
	t = f.deferencePointer(t)

	if t.Kind() != reflect.Struct {
		return "", isVector, nil
	}

	if refs, ok := f.seen[t]; ok {
		if ref, ok := refs[meta.MediaType]; ok {
			return ref.ref, isVector, nil
		}
	}

	info := meta.StructExtractor(t)
	ref := f.makeRefString(info.mediaName)

	f.putSeen(t, meta.MediaType, seen{
		ref:  ref,
		name: info.mediaName,
	})

	schema, err := f.makeSchema(meta, t)
	if err != nil {
		return "", isVector, err
	}

	schema.XML = info.xml

	f.putSeen(t, meta.MediaType, seen{
		ref:    ref,
		name:   info.mediaName,
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

func (f *FactoryStructToSchema) makeSchema(meta mediaMeta, t reflect.Type) (*Schema, error) {
	t = f.deferencePointer(t)

	if t.Kind() != reflect.Struct {
		return nil, nil
	}

	schema := NewSchema()

	for field := range t.Fields() {
		if field.Anonymous || !field.IsExported() {
			continue
		}

		result := meta.FieldExtractor(field)
		if result.ignore {
			continue
		}

		ref, err := f.inferSchema(meta, field.Type)
		if err != nil {
			return nil, err
		}

		ref.Description = field.Tag.Get("description")

		ref = result.decorate(ref)

		isRequired := f.canBeRequired(field) && !result.omitempty
		schema = f.addProperty(schema, result.tag, ref, isRequired)
	}

	return schema, nil
}

func (f *FactoryStructToSchema) addProperty(schema *Schema, name string, property *Schema, isRequired bool) *Schema {
	schema.Properties[name] = property
	if isRequired {
		schema.Required = append(schema.Required, name)
	}
	return schema
}

func (f *FactoryStructToSchema) canBeRequired(field reflect.StructField) bool {
	return field.Type.Kind() != reflect.Pointer &&
		field.Type.Kind() != reflect.Slice &&
		field.Type.Kind() != reflect.Map
}

func (f *FactoryStructToSchema) inferSchema(media mediaMeta, fieldType reflect.Type) (*Schema, error) {
	for fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}

	if schema, ok := f.types[fieldType]; ok {
		return &schema, nil
	}

	switch fieldType.Kind() {
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

func (f *FactoryStructToSchema) inferStruct(media mediaMeta, fieldType reflect.Type) (*Schema, error) {
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

func (f *FactoryStructToSchema) inferArray(media mediaMeta, fieldType reflect.Type) (*Schema, error) {
	itemRef, err := f.inferSchema(media, fieldType.Elem())
	if err != nil {
		return nil, err
	}

	return &Schema{
		Type:  "array",
		Items: itemRef,
	}, nil
}

func (f *FactoryStructToSchema) inferMap(media mediaMeta, fieldType reflect.Type) (*Schema, error) {
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
	for t.Kind() == reflect.Pointer ||
		t.Kind() == reflect.Slice ||
		t.Kind() == reflect.Array {
		t = t.Elem()
	}
	return t
}

func (f *FactoryStructToSchema) isVector(t reflect.Type) bool {
	return t.Kind() == reflect.Slice || t.Kind() == reflect.Array
}

func (f *FactoryStructToSchema) makeRefString(name string) string {
	return fmt.Sprintf("#/components/schemas/%s", name)
}
