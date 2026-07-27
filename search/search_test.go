package search_test

import (
	"path/filepath"
	"testing"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/search"
)

func TestSearchIndexAndQuery(t *testing.T) {
	s := newSearch(t)

	if err := s.Index("test", "catalog", "123", "this is the first doc"); err != nil {
		t.Fatal(err)
	}
	if err := s.Index("test", "catalog", "456", "this is the 2nd doc"); err != nil {
		t.Fatal(err)
	}

	results, err := s.Search("test", "catalog", "first doc")
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, results.IDs, "123")
}

func TestCRMContactSearchAcrossFields(t *testing.T) {
	s := newSearch(t)

	contacts := []struct {
		id     string
		fields map[string]string
	}{
		{
			id: "contact_001",
			fields: map[string]string{
				"firstName": "Denis",
				"lastName":  "St-Pierre",
				"company":   "StaticBackend",
				"email":     "denis@staticbackend.com",
			},
		},
		{
			id: "contact_002",
			fields: map[string]string{
				"firstName": "Jane",
				"lastName":  "Cooper",
				"company":   "Acme Industrial",
				"email":     "jane.cooper@acme.example",
			},
		},
		{
			id: "contact_003",
			fields: map[string]string{
				"firstName": "John",
				"lastName":  "Michaels",
				"company":   "Northwind Logistics",
				"email":     "jmichaels@northwind.example",
			},
		},
	}

	for _, contact := range contacts {
		if err := s.IndexFields("crm_prod", "contacts", contact.id, contact.fields); err != nil {
			t.Fatal(err)
		}
	}

	testCases := []struct {
		name     string
		keywords string
		wantIDs  []string
	}{
		{name: "first name", keywords: "denis", wantIDs: []string{"contact_001"}},
		{name: "last name typo", keywords: "cooper", wantIDs: []string{"contact_002"}},
		{name: "company prefix", keywords: "stat", wantIDs: []string{"contact_001"}},
		{name: "company multi token", keywords: "acme industrial", wantIDs: []string{"contact_002"}},
		{name: "email user", keywords: "jmichaels", wantIDs: []string{"contact_003"}},
		{name: "email domain", keywords: "staticbackend.com", wantIDs: []string{"contact_001"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := s.Search("crm_prod", "contacts", tc.keywords)
			if err != nil {
				t.Fatal(err)
			}
			assertIDs(t, results.IDs, tc.wantIDs...)
		})
	}
}

func TestSearchDelete(t *testing.T) {
	s := newSearch(t)

	if err := s.IndexFields("crm", "contacts", "deleted_contact", map[string]string{
		"firstName": "Alice",
		"lastName":  "Removed",
		"email":     "alice.removed@example.com",
	}); err != nil {
		t.Fatal(err)
	}

	results, err := s.Search("crm", "contacts", "alice")
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, results.IDs, "deleted_contact")

	if err := s.Delete("crm", "contacts", "deleted_contact"); err != nil {
		t.Fatal(err)
	}

	results, err = s.Search("crm", "contacts", "alice")
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, results.IDs)
}

func newSearch(t *testing.T) *search.Search {
	t.Helper()

	pubsub := cache.NewDevCache()

	s, err := search.New(filepath.Join(t.TempDir(), "test.fts"), pubsub)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)

	return s
}

func assertIDs(t *testing.T, got []string, want ...string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected ids %v, got %v", want, got)
	}

	seen := make(map[string]bool, len(got))
	for _, id := range got {
		seen[id] = true
	}
	for _, id := range want {
		if !seen[id] {
			t.Fatalf("expected ids %v, got %v", want, got)
		}
	}
}
