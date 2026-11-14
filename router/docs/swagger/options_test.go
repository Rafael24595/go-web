package swagger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rafael24595/go-assert/assert"
	"github.com/Rafael24595/go-assert/require"
)

const definition = `
openapi: 3.0.3
info:
  title: Media shelf
  description: Local API
`

func withVersion(version string) OpenAPI3ViewerOption {
	return func(data *OpenAPI3) error {
		data.Info.Version = version
		return nil
	}
}

func TestLoad(t *testing.T) {
	t.Run("without options the definition is a valid OpenAPI 3.0", func(t *testing.T) {
		viewer := NewViewer()
		
		require.Nil(t, viewer.Load())

		assert.Equal(t, "3.0.0", viewer.data.OpenAPI)
		assert.Equal(t, "/", viewer.data.Servers[0].URL)
	})

	t.Run("options are applied in order and merge", func(t *testing.T) {
		viewer := NewViewer()

		require.Nil(t, viewer.Load(
			withVersion("1.2.3"),
			nil,
			WithYAML([]byte(definition)),
		))

		assert.Equal(t, "3.0.3", viewer.data.OpenAPI)
		assert.Equal(t, "Media shelf", viewer.data.Info.Title)
		assert.Equal(t, "Local API", viewer.data.Info.Description)
		assert.Equal(t, "1.2.3", viewer.data.Info.Version, "fields absent from the YAML are kept")
	})

	t.Run("a later option overrides an earlier one", func(t *testing.T) {
		viewer := NewViewer()

		require.Nil(t, viewer.Load(
			WithYAML([]byte("info:\n  version: 0.0.1\n")),
			withVersion("1.2.3"),
		))

		assert.Equal(t, "1.2.3", viewer.data.Info.Version)
	})

	t.Run("a YAML file is read from disk", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "swagger.yaml")
		require.Nil(t, os.WriteFile(file, []byte(definition), 0o644))

		viewer := NewViewer()

		require.Nil(t, viewer.Load(WithYAMLFile(file)))

		assert.Equal(t, "Media shelf", viewer.data.Info.Title)
	})

	t.Run("a failing option leaves the viewer unchanged", func(t *testing.T) {
		viewer := NewViewer()
		
		require.Nil(t, viewer.Load(withVersion("1.2.3")))

		assert.NotNil(t, viewer.Load(WithYAMLFile(filepath.Join(t.TempDir(), "missing.yaml"))))
		assert.NotNil(t, viewer.Load(WithYAML([]byte("info: ["))))

		assert.Equal(t, "1.2.3", viewer.data.Info.Version)
	})
}
