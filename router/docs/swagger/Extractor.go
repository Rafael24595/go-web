package swagger

import (
	"reflect"
	"slices"
	"strings"
)

type extractResult struct {
	ignore    bool
	tag       string
	omitempty bool
	ref       *Schema
}

type fieldExtractor func(reflect.StructField, *Schema) extractResult

func voidExtractor(reflect.StructField, *Schema) extractResult {
	return extractResult{}
}

func jsonExtractor(field reflect.StructField, ref *Schema) extractResult {
	name, options, ignore := tagParts(field, "json")
	if ignore {
		return extractResult{ignore: true}
	}

	omitEmpty := slices.Contains(options, "omitempty") ||
		slices.Contains(options, "omitzero")

	return extractResult{
		tag:       name,
		omitempty: omitEmpty,
		ref:       ref,
	}
}

func xmlExtractor(field reflect.StructField, ref *Schema) extractResult {
	name, options, ignore := tagParts(field, "xml")
	if ignore {
		return extractResult{ignore: true}
	}

	if parent, child, ok := strings.Cut(name, ">"); ok {
		name = strings.TrimSpace(parent)
		wrapper := strings.TrimSpace(child)

		if ref.Items != nil {
			ref.Items = defineXmlName(wrapper, ref.Items, false)
		} else {
			ref = defineXmlName(wrapper, ref, false)
		}

		ref = &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				wrapper: ref,
			},
		}
	}

	omitEmpty := slices.Contains(options, "omitempty")
	ref = defineXmlName(name, ref, slices.Contains(options, "attr"))

	return extractResult{
		tag:       name,
		omitempty: omitEmpty,
		ref:       ref,
	}
}

func defineXmlName(name string, ref *Schema, attribute bool) *Schema {
	if ref.Ref != "" {
		refVal := ref.Ref
		ref.Ref = ""

		ref.AllOf = MakeAllOf(AllOfAttributes{
			ALL_OF_REF: refVal,
		})
	}

	ref.XML = &XML{
		Name:      name,
		Attribute: attribute,
	}

	return ref
}

func tagParts(field reflect.StructField, key string) (name string, options []string, ignore bool) {
	value := field.Tag.Get(key)
	if value == "-" {
		return "", nil, true
	}

	name, rest, _ := strings.Cut(value, ",")
	if name == "" {
		name = field.Name
	}

	if rest != "" {
		options = strings.Split(rest, ",")
	}

	return name, options, false
}
