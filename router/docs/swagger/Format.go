package swagger

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"reflect"

	"github.com/Rafael24595/go-web/router/docs"
)

// ErrUnsupportedMediaType is returned when an unsupported media type is requested.
var ErrUnsupportedMediaType = errors.New("swagger: unsupported media type")

type mediaFormat struct {
	mediaType       docs.MediaType
	structExtractor structExtractor
	fieldExtractor  fieldExtractor
	marshaler       reflect.Type
	bytes           Schema
}

var jsonFormat = mediaFormat{
	mediaType:       docs.JSON,
	structExtractor: jsonStructExtractor(),
	fieldExtractor:  jsonFieldExtractor,
	marshaler:       reflect.TypeFor[json.Marshaler](),
	bytes:           Schema{Type: "string", Format: "byte"},
}

var xmlFormat = mediaFormat{
	mediaType:       docs.XML,
	structExtractor: xmlStructExtractor(),
	fieldExtractor:  xmlFieldExtractor,
	marshaler:       reflect.TypeFor[xml.Marshaler](),
	bytes:           Schema{Type: "string"},
}

func formatOf(media docs.MediaType) (mediaFormat, error) {
	switch media {
	case docs.JSON:
		return jsonFormat, nil
	case docs.XML:
		return xmlFormat, nil
	default:
		err := fmt.Errorf("%w: %q", ErrUnsupportedMediaType, media)
		return mediaFormat{}, err
	}
}
