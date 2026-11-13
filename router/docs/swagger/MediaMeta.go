package swagger

import (
	"errors"
	"fmt"

	"github.com/Rafael24595/go-web/router/docs"
)

// ErrUnsupportedMediaType is returned when an unsupported media type is requested.
var ErrUnsupportedMediaType = errors.New("swagger: unsupported media type")

type mediaMeta struct {
	MediaType       docs.MediaType
	StructExtractor structExtractor
	FieldExtractor  fieldExtractor
}

var jsonMeta = mediaMeta{
	MediaType:       docs.JSON,
	StructExtractor: plainStructExtractor("json"),
	FieldExtractor:  jsonFieldExtractor,
}

var xmlMeta = mediaMeta{
	MediaType:       docs.XML,
	StructExtractor: xmlStructExtractor("xml"),
	FieldExtractor:  xmlFieldExtractor,
}

func mediaMetaFromType(media docs.MediaType) (mediaMeta, error) {
	switch media {
	case docs.JSON:
		return jsonMeta, nil
	case docs.XML:
		return xmlMeta, nil
	default:
		err := fmt.Errorf("%w: %q", ErrUnsupportedMediaType, media)
		return mediaMeta{}, err
	}
}
