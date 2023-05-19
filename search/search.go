package search

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
)

type Search struct {
	index bleve.Index
}

type IndexDocument struct {
	DBName string         `json:"dbname"`
	Key    string         `json:"key"`
	ID     string         `json:"id"`
	Text   string         `json:"text"`
	Data   map[string]any `json:"data"`
}

func New(filename string) (*Search, error) {
	s := &Search{}

	if _, err := os.Stat(filename); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		idx, err := createMapping(filename)
		if err != nil {
			return nil, err
		}
		s.index = idx
	} else {
		idx, err := bleve.Open(filename)
		if err != nil {
			return nil, err
		}

		s.index = idx
	}

	return s, nil
}

func createMapping(filename string) (bleve.Index, error) {
	mapping := bleve.NewDocumentMapping()

	dbMap := bleve.NewKeywordFieldMapping()
	mapping.AddFieldMappingsAt("DBName", dbMap)

	keyMap := bleve.NewKeywordFieldMapping()
	mapping.AddFieldMappingsAt("Key", keyMap)

	idMap := bleve.NewKeywordFieldMapping()
	mapping.AddFieldMapping(idMap)

	textMap := bleve.NewTextFieldMapping()
	textMap.Analyzer = "en"
	mapping.AddFieldMappingsAt("Text", textMap)

	idxmap := bleve.NewIndexMapping()
	idxmap.AddDocumentMapping("IndexDocument", mapping)

	return bleve.New(filename, idxmap)
}

func (s *Search) Index(dbName, catalog, id, text string, data map[string]any) error {
	doc := IndexDocument{
		DBName: dbName,
		Key:    catalog,
		ID:     id,
		Text:   text,
		Data:   data,
	}

	docID := fmt.Sprintf("%s-%s-%s", dbName, catalog, id)
	return s.index.Index(docID, doc)
}

func (s *Search) Search(dbName, catalog, keywords string) ([]map[string]any, error) {
	tokens := strings.Split(keywords, " ")

	var queries []query.Query

	for _, keyword := range tokens {
		fq := bleve.NewFuzzyQuery(keyword)
		fq.SetField("Text")

		queries = append(queries, fq)
	}

	conj := bleve.NewConjunctionQuery(queries...)
	if conj == nil {
		return nil, errors.New("conj is nil")
	}

	req := bleve.NewSearchRequest(conj)

	if req == nil {
		return nil, errors.New("wtf? it's nil")
	}

	results, err := s.index.Search(req)
	if err != nil {
		return nil, err
	}

	var docs []map[string]any
	for _, r := range results.Hits {
		doc, ok := r.Fields["data"].(map[string]any)
		if !ok {
			continue
		}
		docs = append(docs, doc)
	}

	return docs, nil
}
