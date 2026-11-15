package swagger

import (
	"cmp"
	"encoding"
	"fmt"
	"reflect"
	"slices"

	"github.com/Rafael24595/go-web/router/docs"
)

var textMarshalerType = reflect.TypeFor[encoding.TextMarshaler]()

type component struct {
	ref    string
	name   string
	schema Schema
}

type property struct {
	name     string
	schema   *Schema
	required bool
	depth    int
	named    bool
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
	embedding := map[reflect.Type]bool{t: true}

	candidates, err := f.collectProperties(format, t, 0, true, embedding)
	if err != nil {
		return nil, err
	}

	schema := NewSchema()

	properties := resolveProperties(candidates)
	for _, property := range properties {
		schema = f.addProperty(
			schema, property.name, property.schema, property.required,
		)
	}

	return schema, nil
}

func (f *schemaFactory) collectProperties(
	format mediaFormat,
	t reflect.Type,
	depth int,
	canRequire bool,
	embedding map[reflect.Type]bool,
) ([]property, error) {
	var properties []property

	for field := range t.Fields() {
		result := format.fieldExtractor(field)
		if result.ignore {
			continue
		}

		if embedded, pointer, ok := promotes(field, result); ok {
			if embedding[embedded] {
				continue
			}

			embedding[embedded] = true

			embeddedDepth := depth + 1
			embeddedRequire := canRequire && !pointer

			promoted, err := f.collectProperties(
				format, embedded, embeddedDepth, embeddedRequire, embedding,
			)

			delete(embedding, embedded)

			if err != nil {
				return nil, err
			}

			properties = append(properties, promoted...)

			continue
		}

		if !field.IsExported() {
			continue
		}

		ref, err := f.inferSchema(format, field.Type)
		if err != nil {
			return nil, err
		}

		ref.Description = field.Tag.Get("description")

		ref = result.decorate(ref)
		isRequired := canRequire && f.canBeRequired(field) && !result.omitempty

		properties = append(properties, property{
			name:     result.name,
			schema:   ref,
			required: isRequired,
			depth:    depth,
			named:    result.named,
		})
	}

	return properties, nil
}

func promotes(field reflect.StructField, result fieldInfo) (reflect.Type, bool, bool) {
	if !field.Anonymous || result.named {
		return nil, false, false
	}

	t := field.Type
	pointer := false

	if t.Kind() == reflect.Pointer {
		t, pointer = t.Elem(), true
	}

	return t, pointer, t.Kind() == reflect.Struct
}

func resolveProperties(candidates []property) []property {
	byName := make(map[string][]property)
	for _, c := range candidates {
		byName[c.name] = append(byName[c.name], c)
	}

	winners := make(map[string]property, len(byName))
	for name, rivals := range byName {
		if winner, ok := dominant(rivals); ok {
			winners[name] = winner
		}
	}

	resolved := make([]property, 0, len(winners))
	for _, c := range candidates {
		winner, ok := winners[c.name]
		if ok && winner == c {
			resolved = append(resolved, c)
		}
	}

	return resolved
}

func dominant(rivals []property) (property, bool) {
	if len(rivals) == 1 {
		return rivals[0], true
	}

	depth := slices.MinFunc(rivals, func(a, b property) int {
		return cmp.Compare(a.depth, b.depth)
	}).depth

	shallowest := slices.DeleteFunc(slices.Clone(rivals), func(p property) bool {
		return p.depth != depth
	})

	if len(shallowest) == 1 {
		return shallowest[0], true
	}

	named := slices.DeleteFunc(shallowest, func(p property) bool {
		return !p.named
	})

	if len(named) == 1 {
		return named[0], true
	}

	return property{}, false
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
