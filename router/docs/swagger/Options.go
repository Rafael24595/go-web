package swagger

import (
	"os"

	"go.yaml.in/yaml/v3"
)

// OpenAPI3ViewerOption sets part of the definition the viewer starts from.
// Options are applied in order: a later one overrides what an earlier one set.
type OpenAPI3ViewerOption func(data *OpenAPI3) error

func applyOptions(options ...OpenAPI3ViewerOption) (*OpenAPI3, error) {
	var err error

	data := &OpenAPI3{
		OpenAPI: "3.0.0",
	}

	for _, option := range options {
		if option == nil {
			continue
		}

		err = option(data)
		if err != nil {
			return data, err
		}
	}

	return data, nil
}

// WithYAML loads an OpenAPI definition from YAML content, for instance one
// embedded with go:embed. Only the fields present in the YAML are set; the
// rest keep the value given by earlier options.
func WithYAML(content []byte) OpenAPI3ViewerOption {
	return func(data *OpenAPI3) error {
		return yaml.Unmarshal(content, data)
	}
}

// WithYAMLFile loads an OpenAPI definition from a YAML file; see WithYAML.
func WithYAMLFile(filename string) OpenAPI3ViewerOption {
	return func(data *OpenAPI3) error {
		content, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		return WithYAML(content)(data)
	}
}
