package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func dbPost(t *testing.T, hf func(http.ResponseWriter, *http.Request), repo string, v interface{}) *http.Response {
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal("error marshaling post data:", err)
	}

	req := httptest.NewRequest("POST", "/db/"+repo, bytes.NewReader(b))
	w := httptest.NewRecorder()

	req.Header.Set("SB-PUBLIC-KEY", pubKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

	h := chain(http.HandlerFunc(hf), auth, withDB)

	h.ServeHTTP(w, req)

	return w.Result()
}

func GetResponseBody(t *testing.T, resp *http.Response) string {
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("error reading response body: ", err)
	}

	return string(b)
}

func TestHasPermission(t *testing.T) {
	reads := make(map[string]permissionLevel)
	reads["tbl_740_"] = permGroup
	reads["tbl_600_"] = permOwner
	reads["tbl"] = permGroup
	reads["tbl_226_"] = permEveryone

	for k, v := range reads {
		if p := readPermission(k); v != p {
			t.Errorf("%s expected read to be %v got %v", k, v, p)
		}
	}

	writes := make(map[string]permissionLevel)
	writes["tbl"] = permOwner
	writes["tbl_760_"] = permGroup
	writes["tbl_662_"] = permEveryone
	writes["tbl_244_"] = permOwner

	for k, v := range writes {
		if p := writePermission(k); v != p {
			t.Errorf("%s expected write to be %v got %v", k, v, p)
		}
	}
}

type Task struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Done    bool      `json:"done"`
	Created time.Time `json:"created"`
}

func TestDBCreate(t *testing.T) {
	task :=
		Task{
			Title:   "item created",
			Created: time.Now(),
		}

	resp := dbPost(t, database.add, "tasks", task)

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var saved Task
	if err := parseBody(resp.Body, &saved); err != nil {
		t.Fatal(err)
	} else if task.Title != saved.Title {
		t.Errorf("expected title to be %s go %s", task.Title, saved.Title)
	}
}
