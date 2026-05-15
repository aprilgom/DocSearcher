package search

import "hwp-searcher/internal/domain"

type documentCodec struct {
	schema domain.IndexSchema
}

func newDocumentCodec(schema domain.IndexSchema) documentCodec {
	return documentCodec{schema: schema}
}

func (c documentCodec) fieldMap(doc domain.IndexedDocument) map[string]string {
	return map[string]string{
		c.schema.ContentField:        doc.Content,
		c.schema.ContentNoSpaceField: doc.ContentNoSpace,
		c.schema.PathField:           string(doc.ID),
		c.schema.RootIDField:         string(doc.RootID),
		c.schema.RelativePathField:   string(doc.RelativePath),
		c.schema.ServerPathField:     doc.ServerPath,
	}
}
