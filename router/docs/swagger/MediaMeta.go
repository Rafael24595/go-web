package swagger

import (
	"github.com/Rafael24595/go-web/router/docs"
)

type mediaMeta struct {
	MediaType docs.MediaType
	Format    string
	Extractor fieldExtractor
}

var jsonMeta = mediaMeta{
	MediaType: docs.JSON,
	Format:    "json",
	Extractor: jsonExtractor,
}

var xmlMeta = mediaMeta{
	MediaType: docs.XML,
	Format:    "xml",
	Extractor: xmlExtractor,
}

var voidMeta = mediaMeta{
	MediaType: "",
	Format:    "",
	Extractor: voidExtractor,
}

func mediaMetaFromType(media docs.MediaType) mediaMeta {
	switch media {
	case docs.JSON:
		return jsonMeta
	case docs.XML:
		return xmlMeta
	default:
		return voidMeta
	}
}
