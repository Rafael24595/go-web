package swagger

import (
	"encoding/xml"
	"reflect"
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
	Dash      string `json:"-,"`
	Lookalike string `json:"omitempty_count" xml:"attribute"`
	Attr      string `xml:",attr"`
	unexposed string
}

type celsius float64

func (c celsius) MarshalJSON() ([]byte, error) { return []byte(`"21.5 C"`), nil }

type reading struct {
	Temperature celsius
	When        time.Time
}

type secret struct{ Token string }

type withIgnored struct {
	Name  string  `json:"name" xml:"name"`
	Cache *secret `json:"-" xml:"-"`
}

type root struct {
	XMLName xml.Name `xml:"shelf"`
	Name    string   `xml:"name"`
}

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
			assert.Equal(t, tt.tag, got.tag)
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
			assert.Equal(t, tt.tag, got.tag)
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
	factory := NewFactoryStructToSchema()

	_, err := factory.MakeSchema(docs.JSON, tagged{})
	require.Nil(t, err)

	schema := factory.Components().Schemas["Json_Swagger_Tagged"]

	_, hidden := schema.Properties["Hidden"]
	_, unexposed := schema.Properties["unexposed"]

	assert.False(t, hidden, "json:\"-\" is not sent")
	assert.False(t, unexposed, "unexported fields are not sent")

	wantRequired := []string{"Plain", "named", "-", "omitempty_count", "Attr"}
	assert.DeepEqual(t, wantRequired, schema.Required)
}

func TestDefineType(t *testing.T) {
	factory := NewFactoryStructToSchema()

	factory.DefineType(
		reflect.TypeFor[celsius](),
		Schema{Type: "string", Format: "temperature"},
	)

	factory.DefineType(
		reflect.TypeFor[time.Time](),
		Schema{Type: "integer", Description: "unix seconds"},
	)

	_, err := factory.MakeSchema(docs.JSON, reading{})
	require.Nil(t, err)

	properties := factory.Components().Schemas["Json_Swagger_Reading"].Properties

	assert.Equal(t, "string", properties["Temperature"].Type)
	assert.Equal(t, "temperature", properties["Temperature"].Format)
	assert.Equal(t, "integer", properties["When"].Type)
}

func TestMakeSchema_IgnoredFieldsAddNoComponents(t *testing.T) {
	media := []docs.MediaType{
		docs.JSON,
		docs.XML,
	}

	for _, media := range media {
		factory := NewFactoryStructToSchema()

		_, err := factory.MakeSchema(media, withIgnored{})
		require.Nil(t, err)

		assert.Size(t, 1, factory.Components().Schemas)
	}
}

func TestMakeSchema_ComponentNames(t *testing.T) {
	factory := NewFactoryStructToSchema()

	_, err := factory.MakeSchema(docs.XML, root{})
	require.Nil(t, err)

	schema, ok := factory.Components().Schemas["Xml_Swagger_Root"]

	require.True(t, ok)
	require.NotNil(t, schema.XML)
	assert.Equal(t, "shelf", schema.XML.Name)

	_, err = factory.MakeSchema(docs.JSON, root{})
	require.Nil(t, err)

	assert.Nil(t, factory.Components().Schemas["Json_Swagger_Root"].XML)
}
