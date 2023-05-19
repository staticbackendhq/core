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

	data := make(map[string]any)
	data["field"] = "a"
	data["ison"] = true

	err = s.Index("test", "catalog", "123", "this is the first doc", data)
	if err != nil {
		t.Fatal(err)
	}

	data["ison"] = false
	err = s.Index("test", "catalog", "123", "this is the 2nd doc", data)
	if err != nil {
		t.Fatal(err)
	}

	results, err := s.Search("test", "catalog", "first doc")
	if err != nil {
		t.Fatal(err)
	} else if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	} else if results[0]["ison"] == true {
		t.Log(results)
		t.Errorf("expected ison to be false, got %v", results[0]["ison"])
	}
}
