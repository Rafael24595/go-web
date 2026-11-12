package swagger

import (
	"errors"
	"os"

	"go.yaml.in/yaml/v3"
)

var (
	// ErrInvalidDstType is returned when the destination value is not a pointer to a struct.
	ErrInvalidDstType = errors.New("dst must be a non-nil pointer to a struct")
	// ErrNilSrcType is returned when the source value is nil.
	ErrNilSrcType = errors.New("src must be a non-nil struct or a pointer to a struct")
	// ErrTypeMismatch is returned when the source and destination values are not of the same type.
	ErrTypeMismatch = errors.New("both values must be structs of the same type")
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
