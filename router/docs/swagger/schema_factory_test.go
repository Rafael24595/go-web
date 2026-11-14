package swagger

import (
	"maps"
	"slices"
	"testing"

	"github.com/Rafael24595/go-assert/assert"
	"github.com/Rafael24595/go-assert/require"
	"github.com/Rafael24595/go-web/router/docs"
)

func TestMakeSchema_EmbeddedStructs(t *testing.T) {
	type audit struct {
		Created string `json:"created"`
		Updated string `json:"updated,omitempty"`
	}

	type owner struct {
		Owner string `json:"owner"`
	}

	type hidden struct {
		Secret string `json:"secret"`
	}

	type labels struct {
		Name string `json:"name"`
	}

	type document struct {
		audit
		*owner
		Meta   labels `json:"meta"`
		hidden `json:"-"`
		Name   string `json:"name"`
		Title  string `json:"title"`
	}

	factory := newSchemaFactory()

	_, err := factory.makeSchema(docs.JSON, document{})
	require.Nil(t, err)

	schema := factory.components().Schemas["Json_Swagger_Document"]

	names := slices.Sorted(maps.Keys(schema.Properties))

	assert.DeepEqual(t, []string{"created", "meta", "name", "owner", "title", "updated"}, names)
	assert.DeepEqual(t, []string{"created", "meta", "name", "title"}, schema.Required)
}

func TestMakeSchema_EmbeddedConflicts(t *testing.T) {
	type left struct {
		ID string
	}

	type right struct {
		ID string
	}

	type ambiguous struct {
		left
		right
	}

	type plainID struct {
		ID string `description:"untagged"`
	}

	type taggedID struct {
		ID string `json:"ID" description:"tagged"`
	}

	type resolvedByTag struct {
		plainID
		taggedID
	}

	type selfEmbedding struct {
		//nolint:unused
		*selfEmbedding
		Name string `json:"name"`
	}

	properties := func(payload any, component string) []string {
		t.Helper()

		factory := newSchemaFactory()

		_, err := factory.makeSchema(docs.JSON, payload)
		require.Nil(t, err)

		return slices.Sorted(maps.Keys(factory.components().Schemas[component].Properties))
	}

	assert.Size(t, 0, properties(ambiguous{}, "Json_Swagger_Ambiguous"))
	assert.DeepEqual(t, []string{"ID"}, properties(resolvedByTag{}, "Json_Swagger_ResolvedByTag"))

	factory := newSchemaFactory()

	_, err := factory.makeSchema(docs.JSON, resolvedByTag{})
	require.Nil(t, err)

	id := factory.components().Schemas["Json_Swagger_ResolvedByTag"].Properties["ID"]

	assert.Equal(t, "tagged", id.Description)
	assert.DeepEqual(t, []string{"name"}, properties(selfEmbedding{}, "Json_Swagger_SelfEmbedding"))
}
