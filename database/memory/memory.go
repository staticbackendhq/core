package memory

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/database"
)

/*const (
	FieldID        = "id"
	FieldAccountID = "accountId"
	FieldOwnerID   = "ownerId"
)*/

var collectionNotFoundErr = errors.New("collection not found")

func init() {
	gob.Register(map[string]any{})
	gob.Register([]any{})
	gob.Register(time.Time{})
}

type Memory struct {
	DB              map[string]map[string][]byte
	PublishDocument cache.PublishDocumentEvent
}

func New(pubdoc cache.PublishDocumentEvent) database.Persister {
	db := make(map[string]map[string][]byte)

	if err := initDB(db); err != nil {
		log.Fatal(err)
	}

	return &Memory{DB: db, PublishDocument: pubdoc}
}

func initDB(db map[string]map[string][]byte) error {
	db["sb_customers"] = make(map[string][]byte)
	db["sb_apps"] = make(map[string][]byte)
	return nil
}

func (m *Memory) NewID() string {
	return uuid.NewString()
}

func (m *Memory) Ping() error {
	return nil
}

func (m *Memory) CreateIndex(dbName, col, field string) error {
	return nil
}

func mustEnc(v any) []byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

func mustDec(b []byte, v any) error {
	return gob.NewDecoder(bytes.NewReader(b)).Decode(v)
}

func create[T any](m *Memory, dbName, col, id string, v T) error {
	key := fmt.Sprintf("%s_%s", dbName, col)

	repo, ok := m.DB[key]
	if !ok {
		repo = make(map[string][]byte)
	}

	repo[id] = mustEnc(v)

	m.DB[key] = repo
	return nil
}

func getByID[T any](m *Memory, dbName, col, id string, v T) error {
	key := fmt.Sprintf("%s_%s", dbName, col)

	repo, ok := m.DB[key]
	if !ok {
		return collectionNotFoundErr
	}

	b, ok := repo[id]
	if !ok {
		return errors.New("document not found")
	} else if err := mustDec(b, v); err != nil {
		return err
	}
	return nil
}

func all[T any](m *Memory, dbName, col string) (list []T, err error) {
	key := fmt.Sprintf("%s_%s", dbName, col)

	repo, ok := m.DB[key]
	if !ok {
		return nil, collectionNotFoundErr
	}

	for _, v := range repo {
		var li T
		if err = mustDec(v, &li); err != nil {
			return
		}

		list = append(list, li)
	}

	return
}

func filter[T any](list []T, fn func(x T) bool) []T {
	var results []T
	for _, item := range list {
		if fn(item) {
			results = append(results, item)
		}
	}

	return results
}

func filterByClauses(list []map[string]any, filter map[string]any) (filtered []map[string]any) {
	for _, doc := range list {
		matches := 0
		for k, v := range filter {
			op, field := extractOperatorAndValue(k)
			switch op {
			case "=":
				if equal(doc[field], v) {
					matches++
				}
			case "!=":
				if notEqual(doc[field], v) {
					matches++
				}
			case ">":
				if greater(doc[field], v) {
					matches++
				}
			case "<":
				if lower(doc[field], v) {
					matches++
				}
			case ">=":
				if greaterThanEqual(doc[field], v) {
					matches++
				}
			case "<=":
				if lowerThanEqual(doc[field], v) {
					matches++
				}
			}
		}

		if matches == len(filter) {
			filtered = append(filtered, doc)
		}
	}
	return
}

func sortSlice[T any](list []T, fn func(a, b T) bool) []T {
	sort.Slice(list, func(i, j int) bool {
		return fn(list[i], list[j])
	})
	return list
}

/*
func create_bolt[T any](m *Memory, dbName, col, id string, v T) error {
	bucketName := fmt.Sprintf("%s_%s", dbName, col)

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return m.DB.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}

		return b.Put([]byte(id), data)
	})
}

func getByID_bolt[T any](m *Memory, dbName, col, id string, v T) error {
	bucketName := fmt.Sprintf("%s_%s", dbName, col)

	var data []byte
	err := m.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			if bytes.Equal(k, []byte(id)) {
				data = v
				return nil
			}
		}
		return fmt.Errorf("cannot find id: %s in bucket: %s", id, bucketName)
	})
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

func all_bolt[T any](m *Memory, dbName, col string) (list []T, err error) {
	bucketName := fmt.Sprintf("%s_%s", dbName, col)
	err = m.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var m T
			if err := json.Unmarshal(v, m); err != nil {
				return err
			}

			list = append(list, m)
		}
		return nil
	})

	return
}
*/
