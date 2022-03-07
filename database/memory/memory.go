package memory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"

	"github.com/boltdb/bolt"
	"github.com/google/uuid"
	"github.com/staticbackendhq/core/internal"
)

const (
	FieldID        = "id"
	FieldAccountID = "accountId"
	FieldOwnerID   = "ownerId"
)

type Memory struct {
	DB              *bolt.DB
	PublishDocument internal.PublishDocumentEvent
}

func New(db *bolt.DB, pubdoc internal.PublishDocumentEvent) internal.Persister {
	err := db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte("sb_customers")); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte("sb_apps")); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	return &Memory{DB: db, PublishDocument: pubdoc}
}

func (m *Memory) NewID() string {
	return uuid.NewString()
}

func (m *Memory) Ping() error {
	return nil
}

func (m *Memory) create(dbName, col, id string, v interface{}) error {
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

func (m *Memory) getByID(dbName, col, id string, v interface{}) error {
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
		return fmt.Errorf("cannot find id: %d in bucket: %s", id, bucketName)
	})
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

func (m *Memory) all(dbName, col string) ([]map[string]interface{}, error) {
	bucketName := fmt.Sprintf("%s_%s", dbName, col)
	var list []map[string]interface{}
	err := m.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var m map[string]interface{}
			if err := json.Unmarshal(v, m); err != nil {
				return err
			}

			list = append(list, m)
		}
		return fmt.Errorf("cannot find id: %d in bucket: %s", id, bucketName)
	})
	if err != nil {
		return nil, err
	}

	return list, nil
}

type FilterParam struct {
	Field string
	Op    string
	Value interface{}
}

func (m *Memory) filter(list []map[string]interface{}, param FilterParam) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	var match bool
	for _, item := range list {
		v := item[param.Field]

		switch param.Op {
		case "=":
			match = param.Value == v
		case "!=":
			match = param.Value != v
		default:
			match = false
		}

		if match {
			results = append(results, item)
		}
	}

	return results, nil
}
