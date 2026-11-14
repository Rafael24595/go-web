package swagger

import (
	"encoding/json"
	"encoding/xml"
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Rafael24595/go-assert/assert"
	"github.com/Rafael24595/go-assert/require"
	"github.com/Rafael24595/go-web/router/docs"
)

type tagged struct {
	Plain     string
	Named     string `json:"named" xml:"named"`
	Omit      string `json:",omitempty" xml:",omitempty"`
	Zero      string `json:"zero,omitzero"`
	Hidden    string `json:"-" xml:"-"`
	//nolint:staticcheck
	Dash      string `json:"-,"`
	Lookalike string `json:"omitempty_count" xml:"attribute"`
	Attr      string `xml:",attr"`
	//nolint:unused
	unexposed string
}

type celsius float64

func (c celsius) MarshalJSON() ([]byte, error) { return []byte(`"21.5 C"`), nil }

func field(t *testing.T, name string) reflect.StructField {
	t.Helper()

	f, ok := reflect.TypeFor[tagged]().
		FieldByName(name)

	require.True(t, ok, "no field %s", name)
	return f
}

func TestJsonExtractor(t *testing.T) {
	tests := []struct {
		field     string
		tag       string
		omitEmpty bool
		ignore    bool
	}{
		{
			field: "Plain",
			tag:   "Plain",
		},
		{
			field: "Named",
			tag:   "named",
		},
		{
			field:     "Omit",
			tag:       "Omit",
			omitEmpty: true,
		},
		{
			field:     "Zero",
			tag:       "zero",
			omitEmpty: true},

		{
			field:  "Hidden",
			ignore: true,
		},
		{
			field: "Dash",
			tag:   "-",
		},
		{
			field: "Lookalike",
			tag:   "omitempty_count",
		},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			got := jsonFieldExtractor(field(t, tt.field))

			assert.Equal(t, tt.ignore, got.ignore)
			assert.Equal(t, tt.tag, got.name)
			assert.Equal(t, tt.omitEmpty, got.omitempty)
		})
	}
}

func TestXmlExtractor(t *testing.T) {
	tests := []struct {
		field     string
		tag       string
		omitEmpty bool
		attribute bool
		ignore    bool
	}{
		{
			field: "Plain",
			tag:   "Plain",
		},
		{
			field: "Named",
			tag:   "named",
		},
		{
			field:     "Omit",
			tag:       "Omit",
			omitEmpty: true,
		},
		{
			field:  "Hidden",
			ignore: true,
		},
		{
			field: "Lookalike",
			tag:   "attribute",
		},
		{
			field:     "Attr",
			tag:       "Attr",
			attribute: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			got := xmlFieldExtractor(field(t, tt.field))

			assert.Equal(t, tt.ignore, got.ignore)
			assert.Equal(t, tt.tag, got.name)
			assert.Equal(t, tt.omitEmpty, got.omitempty)

			if !tt.ignore {
				require.NotNil(t, got.decorate)

				ref := got.decorate(&Schema{Type: "string"})
				require.NotNil(t, ref.XML)

				assert.Equal(t, tt.tag, ref.XML.Name)
				assert.Equal(t, tt.attribute, ref.XML.Attribute)
			}
		})
	}
}

func TestMakeSchema_FollowsTheEncoders(t *testing.T) {
	factory := newSchemaFactory()

	_, err := factory.makeSchema(docs.JSON, tagged{})
	require.Nil(t, err)

	schema := factory.components().Schemas["Json_Swagger_Tagged"]

	_, hidden := schema.Properties["Hidden"]
	_, unexposed := schema.Properties["unexposed"]

	assert.False(t, hidden, "json:\"-\" is not sent")
	assert.False(t, unexposed, "unexported fields are not sent")

	wantRequired := []string{"Plain", "named", "-", "omitempty_count", "Attr"}
	assert.DeepEqual(t, wantRequired, schema.Required)
}

func TestDefineType(t *testing.T) {
	type reading struct {
		Temperature celsius
		When        time.Time
	}

	factory := newSchemaFactory()

	factory.defineType(
		reflect.TypeFor[celsius](),
		Schema{Type: "string", Format: "temperature"},
	)

	factory.defineType(
		reflect.TypeFor[time.Time](),
		Schema{Type: "integer", Description: "unix seconds"},
	)

	_, err := factory.makeSchema(docs.JSON, reading{})
	require.Nil(t, err)

	properties := factory.components().Schemas["Json_Swagger_Reading"].Properties

	assert.Equal(t, "string", properties["Temperature"].Type)
	assert.Equal(t, "temperature", properties["Temperature"].Format)
	assert.Equal(t, "integer", properties["When"].Type)
}

func TestMakeSchema_IgnoredFieldsAddNoComponents(t *testing.T) {
	type secret struct {
		Token string
	}

	type withIgnored struct {
		Name  string  `json:"name" xml:"name"`
		Cache *secret `json:"-" xml:"-"`
	}

	media := []docs.MediaType{
		docs.JSON,
		docs.XML,
	}

	for _, media := range media {
		factory := newSchemaFactory()

		_, err := factory.makeSchema(media, withIgnored{})
		require.Nil(t, err)

		assert.Size(t, 1, factory.components().Schemas)
	}
}

func TestMakeSchema_ComponentNames(t *testing.T) {
	type root struct {
		XMLName xml.Name `xml:"shelf"`
		Name    string   `xml:"name"`
	}

	factory := newSchemaFactory()

	_, err := factory.makeSchema(docs.XML, root{})
	require.Nil(t, err)

	schema, ok := factory.components().Schemas["Xml_Swagger_Root"]

	require.True(t, ok)
	require.NotNil(t, schema.XML)
	assert.Equal(t, "shelf", schema.XML.Name)

	_, err = factory.makeSchema(docs.JSON, root{})
	require.Nil(t, err)

	assert.Nil(t, factory.components().Schemas["Json_Swagger_Root"].XML)
}

func TestMakeSchema_TypesThatEncodeThemselves(t *testing.T) {
	type encoded struct {
		When     time.Time
		Maybe    *time.Time
		Raw      json.RawMessage
		Data     []byte
		Address  netip.Addr
		Small    int8
		Anything any
	}

	property := func(media docs.MediaType, component, name string) Schema {
		t.Helper()

		factory := newSchemaFactory()

		_, err := factory.makeSchema(media, encoded{})
		require.Nil(t, err)

		return *factory.components().Schemas[component].Properties[name]
	}

	components := map[docs.MediaType]string{
		docs.JSON: "Json_Swagger_Encoded",
		docs.XML:  "Xml_Swagger_Encoded",
	}

	for media, component := range components {
		assert.Equal(t, "string", property(media, component, "Address").Type)
		assert.Equal(t, "integer", property(media, component, "Small").Type)
		assert.Equal(t, "", property(media, component, "Anything").Type)
	}

	assert.Equal(t, "", property(docs.JSON, components[docs.JSON], "When").Type)
	assert.Equal(t, "string", property(docs.XML, components[docs.XML], "When").Type)
	assert.Equal(t, "string", property(docs.XML, components[docs.XML], "Maybe").Type)

	assert.Equal(t, "byte", property(docs.JSON, components[docs.JSON], "Data").Format)
	assert.Equal(t, "", property(docs.XML, components[docs.XML], "Data").Format)
	assert.Equal(t, "", property(docs.JSON, components[docs.JSON], "Raw").Type)
}

func TestMakeSchema_Payloads(t *testing.T) {
	type item struct {
		Name string `json:"name"`
	}

	ref := "#/components/schemas/Json_Swagger_Item"

	tests := []struct {
		name    string
		payload any
		want    Schema
	}{
		{
			name:    "a struct is referenced",
			payload: item{},
			want:    Schema{Ref: ref},
		},
		{
			name:    "a pointer is its element",
			payload: &item{},
			want:    Schema{Ref: ref},
		},
		{
			name:    "a slice of structs",
			payload: []item{},
			want: Schema{
				Type:  "array",
				Items: &Schema{Ref: ref},
			},
		},
		{
			name:    "a slice of strings",
			payload: []string{},
			want: Schema{
				Type: "array",
				Items: &Schema{
					Type: "string",
				},
			},
		},
		{
			name:    "a map",
			payload: map[string]item{},
			want: Schema{
				Type:                 "object",
				AdditionalProperties: &Schema{Ref: ref},
			},
		},
		{
			name:    "a basic type",
			payload: "",
			want: Schema{
				Type: "string",
			},
		},
		{
			name:    "nil is any value",
			payload: nil,
			want:    Schema{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newSchemaFactory().makeSchema(docs.JSON, tt.payload)

			require.Nil(t, err)
			assert.DeepEqual(t, tt.want, *got)
		})
	}
}

func TestMakeSchema_DefinedTypeAsPayload(t *testing.T) {
	schema := Schema{
		Type:   "string",
		Format: "date-time",
	}

	factory := newSchemaFactory().
		defineType(reflect.TypeFor[time.Time](), schema)

	got, err := factory.makeSchema(docs.JSON, time.Time{})
	require.Nil(t, err)

	assert.Equal(t, "date-time", got.Format)
}

func TestMakeSchema_RecursiveStruct(t *testing.T) {
	type node struct {
		Name     string `json:"name"`
		Children []node `json:"children"`
	}

	factory := newSchemaFactory()

	_, err := factory.makeSchema(docs.JSON, node{})
	require.Nil(t, err)

	children := factory.components().Schemas["Json_Swagger_Node"].Properties["children"]
	assert.Equal(t, "#/components/schemas/Json_Swagger_Node", children.Items.Ref)
}

func TestMakeSchema_ComponentNamesAreUnique(t *testing.T) {
	type pair struct {
		First  struct{ A string } `json:"first"`
		Second struct{ B int }    `json:"second"`
	}

	factory := newSchemaFactory()

	_, err := factory.makeSchema(docs.JSON, pair{})
	require.Nil(t, err)

	schemas := factory.components().Schemas
	require.Size(t, 3, schemas)

	properties := schemas["Json_Swagger_Pair"].Properties

	first := strings.TrimPrefix(properties["first"].Ref, "#/components/schemas/")
	second := strings.TrimPrefix(properties["second"].Ref, "#/components/schemas/")

	assert.NotEqual(t, first, second)

	assert.Equal(t, "string", schemas[first].Properties["A"].Type)
	assert.Equal(t, "integer", schemas[second].Properties["B"].Type)
}
