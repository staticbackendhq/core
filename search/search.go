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
	DBName string `json:"dbname"`
	Key    string `json:"key"`
	Text   string `json:"text"`
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
	mapping.AddFieldMappingsAt("dbname", dbMap)

	keyMap := bleve.NewKeywordFieldMapping()
	mapping.AddFieldMappingsAt("key", keyMap)

	textMap := bleve.NewTextFieldMapping()
	textMap.Analyzer = "en"
	mapping.AddFieldMappingsAt("text", textMap)

	idxmap := bleve.NewIndexMapping()
	idxmap.AddDocumentMapping("IndexDocument", mapping)

	return bleve.New(filename, idxmap)
}

func (s *Search) Index(dbName, col, id, text string) error {
	doc := IndexDocument{
		DBName: dbName,
		Key:    col,
		Text:   text,
	}

	docID := fmt.Sprintf("%s_%s_%s", dbName, col, id)
	return s.index.Index(docID, doc)
}

type SearchResult struct {
	DBName string
	Col    string
	IDs    []string
}

func (s *Search) Search(dbName, col, keywords string) (SearchResult, error) {
	sr := SearchResult{DBName: dbName, Col: col}

	tokens := strings.Split(keywords, " ")

	var queries []query.Query

	dbQry := bleve.NewTermQuery(dbName)
	dbQry.SetField("dbname")

	colQry := bleve.NewTermQuery(col)
	colQry.SetField("key")

	queries = append(queries, dbQry)
	queries = append(queries, colQry)

	for _, keyword := range tokens {
		fq := bleve.NewFuzzyQuery(keyword)
		fq.SetField("text")

		queries = append(queries, fq)
	}

	conj := bleve.NewConjunctionQuery(queries...)
	if conj == nil {
		return sr, errors.New("conj is nil")
	}

	req := bleve.NewSearchRequest(conj)

	if req == nil {
		return sr, errors.New("wtf? it's nil")
	}

	results, err := s.index.Search(req)
	if err != nil {
		return sr, err
	}

	for _, r := range results.Hits {
		sr.IDs = append(sr.IDs, r.ID)
	}

	return sr, nil
}
