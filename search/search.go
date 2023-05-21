package search

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/model"
)

const (
	ChannelIndexEvent = "sys-fts"
)

type Search struct {
	pubsub cache.Volatilizer
	index  bleve.Index
}

type IndexDocument struct {
	ID     string `json:"id"`
	DBName string `json:"dbname"`
	Key    string `json:"key"`
	Text   string `json:"text"`
}

func New(filename string, pubsub cache.Volatilizer) (*Search, error) {
	s := &Search{pubsub: pubsub}

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

	go s.setupIndexEvent()
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
		ID:     id,
		DBName: dbName,
		Key:    col,
		Text:   text,
	}

	b, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	msg := model.Command{
		SID:           "system",
		Type:          "system",
		Data:          string(b),
		Channel:       ChannelIndexEvent,
		Token:         "system",
		IsSystemEvent: true,
	}

	return s.pubsub.Publish(msg)
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
		parts := strings.Split(r.ID, "_")
		if len(parts) != 3 {
			continue
		}

		sr.IDs = append(sr.IDs, parts[2])
	}

	return sr, nil
}

func (s *Search) setupIndexEvent() {
	receiver := make(chan model.Command)
	close := make(chan bool)

	go s.pubsub.Subscribe(receiver, "system", ChannelIndexEvent, close)

	for {
		select {
		case msg := <-receiver:
			go s.receivedIndexEvent(msg.Data)
		case <-close:
			break
		}
	}
}

func (s *Search) receivedIndexEvent(data string) {
	var doc IndexDocument
	if err := json.Unmarshal([]byte(data), &doc); err != nil {
		log.Println(err)
		return
	}

	docID := fmt.Sprintf("%s_%s_%s", doc.DBName, doc.Key, doc.ID)
	if err := s.index.Index(docID, doc); err != nil {
		log.Println(err)
	}
}
