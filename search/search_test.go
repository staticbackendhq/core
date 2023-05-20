package search_test

import (
	"fmt"
	"testing"

	"github.com/staticbackendhq/core/search"
)

func TestSearchIndexAndQuery(t *testing.T) {
	s, err := search.New("testdata/test.fts")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("starting indexing")

	err = s.Index("test", "catalog", "123", "this is the first doc")
	if err != nil {
		t.Fatal(err)
	}

	err = s.Index("test", "catalog", "456", "this is the 2nd doc")
	if err != nil {
		t.Fatal(err)
	}

	results, err := s.Search("test", "catalog", "first doc")
	if err != nil {
		t.Fatal(err)
	} else if len(results.IDs) != 1 {
		t.Errorf("expected 1 result, got %d", len(results.IDs))
	} else if results.IDs[0] != "test_catalog_123" {
		t.Log(results)
		t.Errorf("expected id to be test_catalog_123 got %s", results.IDs[0])
	}
}
