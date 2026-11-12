package swagger

import (
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
			got := jsonExtractor(
				field(t, tt.field), &Schema{Type: "string"},
			)

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
			got := xmlExtractor(
				field(t, tt.field), &Schema{Type: "string"},
			)

			assert.Equal(t, tt.ignore, got.ignore)
			assert.Equal(t, tt.tag, got.tag)
			assert.Equal(t, tt.omitEmpty, got.omitempty)

			if !tt.ignore {
				require.NotNil(t, got.ref.XML)
				assert.Equal(t, tt.attribute, got.ref.XML.Attribute)
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

	assert.DeepEqual(t, []string{"Plain", "named", "-", "omitempty_count", "Attr"}, schema.Required)
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
	assert.Equal(t, "integer", properties["When"].Type, "built-in definitions can be replaced")
}
