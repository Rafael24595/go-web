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
	component string
	xml       *XML
}

func plainStructExtractor(format string) structExtractor {
	return func(t reflect.Type) structInfo {
		name := structName(t)
		mediaName := makeStructName(format, t.PkgPath(), name)

		return structInfo{
			component: mediaName,
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

func jsonStructExtractor() structExtractor {
	return plainStructExtractor("json")
}

func xmlStructExtractor() structExtractor {
	plainExtractor := plainStructExtractor("xml")
	return func(t reflect.Type) structInfo {
		info := plainExtractor(t)

		name := structName(t)
		if root, ok := xmlRootName(t); ok {
			name = root
		}

		info.xml = &XML{
			Name:    name,
			Wrapped: true,
		}

		return info
	}
}

func xmlRootName(t reflect.Type) (string, bool) {
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
	name      string
	named     bool
	omitempty bool
	decorate  fieldDecorator
}

type tagInfo struct {
	name    string
	named   bool
	options []string
	ignore  bool
}

type fieldExtractor func(reflect.StructField) fieldInfo

func jsonFieldExtractor(field reflect.StructField) fieldInfo {
	info := parseTag(field, "json")
	if info.ignore {
		return fieldInfo{ignore: true}
	}

	omitEmpty := slices.Contains(info.options, "omitempty") ||
		slices.Contains(info.options, "omitzero")

	return fieldInfo{
		name:      info.name,
		named:     info.named,
		omitempty: omitEmpty,
		decorate:  identityDecorator,
	}
}

func xmlFieldExtractor(field reflect.StructField) fieldInfo {
	if field.Type == reflect.TypeFor[xml.Name]() {
		return fieldInfo{ignore: true}
	}

	info := parseTag(field, "xml")
	if info.ignore {
		return fieldInfo{ignore: true}
	}

	wrapper := ""
	if parent, child, ok := strings.Cut(info.name, ">"); ok {
		info.name = strings.TrimSpace(parent)
		wrapper = strings.TrimSpace(child)
	}

	omitEmpty := slices.Contains(info.options, "omitempty")
	decorator := xmlFieldDecorator(info.name, wrapper, info.options)

	return fieldInfo{
		name:      info.name,
		named:     info.named,
		omitempty: omitEmpty,
		decorate:  decorator,
	}
}

func withXmlName(name string, ref *Schema, attribute bool) *Schema {
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

func parseTag(field reflect.StructField, key string) tagInfo {
	value := field.Tag.Get(key)
	if value == "-" {
		return tagInfo{
			ignore: true,
		}
	}

	name, rest, _ := strings.Cut(value, ",")
	named := name != ""

	if !named {
		name = field.Name
	}

	var options []string
	if rest != "" {
		options = strings.Split(rest, ",")
	}

	return tagInfo{
		name:    name,
		named:   named,
		options: options,
	}
}

type fieldDecorator func(*Schema) *Schema

func identityDecorator(ref *Schema) *Schema {
	return ref
}

func xmlFieldDecorator(name, wrapper string, options []string) fieldDecorator {
	return func(ref *Schema) *Schema {
		if wrapper != "" {
			if ref.Items != nil {
				ref.Items = withXmlName(wrapper, ref.Items, false)
			} else {
				ref = withXmlName(wrapper, ref, false)
			}

			ref = &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					wrapper: ref,
				},
			}
		}

		attribute := slices.Contains(options, "attr")
		return withXmlName(name, ref, attribute)
	}
}
