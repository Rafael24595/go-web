package swagger

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type structExtractor func(t reflect.Type) structInfo

type structInfo struct {
	mediaName string
	xml       *XML
}

func plainStructExtractor(format string) structExtractor {
	return func(t reflect.Type) structInfo {
		name := structName(t)
		mediaName := makeStructName(format, t.PkgPath(), name)

		return structInfo{
			mediaName: mediaName,
		}
	}
}

func structName(t reflect.Type) string {
	if name := t.Name(); name != "" {
		return name
	}
	return "Anon"
}

func makeStructName(format, pkg, name string) string {
	fragments := strings.Split(pkg, "/")
	pkgFormat := fragments[len(fragments)-1]

	mediaFormat := format
	nameFormat := name

	if mediaFormat != "" {
		caser := cases.Title(language.Und, cases.NoLower)

		mediaFormat = caser.String(mediaFormat)
		pkgFormat = caser.String(pkgFormat)
		nameFormat = caser.String(nameFormat)
	}

	if pkgFormat == "" {
		return fmt.Sprintf("%s_%s", mediaFormat, nameFormat)
	}

	return fmt.Sprintf("%s_%s_%s", mediaFormat, pkgFormat, nameFormat)
}

func xmlStructExtractor(format string) structExtractor {
	plainExtractor := plainStructExtractor(format)
	return func(t reflect.Type) structInfo {
		info := plainExtractor(t)

		name := structName(t)
		if root, ok := hasXmlRoot(t); ok {
			name = root
		}

		info.xml = &XML{
			Name:    name,
			Wrapped: true,
		}

		return info
	}
}

func hasXmlRoot(t reflect.Type) (string, bool) {
	for field := range t.Fields() {
		if field.Type != reflect.TypeFor[xml.Name]() {
			continue
		}

		attribute := field.Tag.Get("xml")
		tag, _, _ := strings.Cut(attribute, ",")

		return tag, tag != "" && tag != "-"
	}

	return "", false
}

type fieldInfo struct {
	ignore    bool
	tag       string
	omitempty bool
	decorate  fieldDecorator
}

type fieldExtractor func(reflect.StructField) fieldInfo

func jsonFieldExtractor(field reflect.StructField) fieldInfo {
	name, options, ignore := tagParts(field, "json")
	if ignore {
		return fieldInfo{ignore: true}
	}

	omitEmpty := slices.Contains(options, "omitempty") ||
		slices.Contains(options, "omitzero")

	return fieldInfo{
		tag:       name,
		omitempty: omitEmpty,
		decorate:  voidFieldDecorator,
	}
}

func xmlFieldExtractor(field reflect.StructField) fieldInfo {
	if field.Type == reflect.TypeFor[xml.Name]() {
		return fieldInfo{ignore: true}
	}

	name, options, ignore := tagParts(field, "xml")
	if ignore {
		return fieldInfo{ignore: true}
	}

	wrapper := ""
	if parent, child, ok := strings.Cut(name, ">"); ok {
		name = strings.TrimSpace(parent)
		wrapper = strings.TrimSpace(child)
	}

	omitEmpty := slices.Contains(options, "omitempty")
	decorator := xmlFieldDecorator(name, wrapper, options)

	return fieldInfo{
		tag:       name,
		omitempty: omitEmpty,
		decorate:  decorator,
	}
}

func defineXmlFieldName(name string, ref *Schema, attribute bool) *Schema {
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

type fieldDecorator func(*Schema) *Schema

func voidFieldDecorator(ref *Schema) *Schema {
	return ref
}

func xmlFieldDecorator(name, wrapper string, options []string) fieldDecorator {
	return func(ref *Schema) *Schema {
		if wrapper != "" {
			if ref.Items != nil {
				ref.Items = defineXmlFieldName(wrapper, ref.Items, false)
			} else {
				ref = defineXmlFieldName(wrapper, ref, false)
			}

			ref = &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					wrapper: ref,
				},
			}
		}

		attribute := slices.Contains(options, "attr")
		return defineXmlFieldName(name, ref, attribute)
	}
}
