package staticbackend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/model"
)

// dbReq post on behalf of adminToken by default (use:
// params[0] true for root)
// params[1] true for Content-Type application/x-www-form-urlencoded
func dbReq(t *testing.T, hf func(http.ResponseWriter, *http.Request), method, path string, v interface{}, params ...bool) *http.Response {
	if params == nil {
		params = make([]bool, 2)
	}

	for i := len(params); i < 2; i++ {
		params = append(params, false)
	}

	var payload []byte
	if params[1] {
		val, ok := v.(url.Values)
		if !ok {
			t.Fatal("expected v to be url.Value")
		}

		payload = []byte(val.Encode())
	} else {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal("error marshaling post data:", err)
		}

		payload = b
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	w := httptest.NewRecorder()

	if params[1] {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req.Header.Add("Content-Type", "application/json")
	}

	req.Header.Set("SB-PUBLIC-KEY", pubKey)

	tok := adminToken
	if params[0] {
		tok = rootToken
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tok))

	stdAuth := []middleware.Middleware{
		middleware.WithDB(backend.DB, backend.Cache, getStripePortalURL),
		middleware.RequireAuth(backend.DB, backend.Cache),
	}
	if params[0] {
		stdAuth = []middleware.Middleware{
			middleware.WithDB(backend.DB, backend.Cache, getStripePortalURL),
			middleware.RequireRoot(backend.DB, backend.Cache),
		}
	}
	h := middleware.Chain(http.HandlerFunc(hf), stdAuth...)

	h.ServeHTTP(w, req)

	return w.Result()
}

func GetResponseBody(t *testing.T, resp *http.Response) string {
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("error reading response body: ", err)
	}

	return fmt.Sprintf("HTTP Status:%s Error:%s", resp.Status, string(b))
}

func TestHasPermission(t *testing.T) {
	reads := make(map[string]internal.PermissionLevel)
	reads["tbl_740_"] = internal.PermGroup
	reads["tbl_600_"] = internal.PermOwner
	reads["tbl"] = internal.PermGroup
	reads["tbl_226_"] = internal.PermEveryone

	for k, v := range reads {
		if p := internal.ReadPermission(k); v != p {
			t.Errorf("%s expected read to be %v got %v", k, v, p)
		}
	}

	writes := make(map[string]internal.PermissionLevel)
	writes["tbl"] = internal.PermOwner
	writes["tbl_760_"] = internal.PermGroup
	writes["tbl_662_"] = internal.PermEveryone
	writes["tbl_244_"] = internal.PermOwner

	for k, v := range writes {
		if p := internal.WritePermission(k); v != p {
			t.Errorf("%s expected write to be %v got %v", k, v, p)
		}
	}
}

type Task struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Done    bool      `json:"done"`
	Created time.Time `json:"created"`
	Count   int       `json:"count"`
}

func TestDBCreate(t *testing.T) {
	task :=
		Task{
			Title:   "item created",
			Created: time.Now(),
		}

	resp := dbReq(t, db.add, "POST", "/db/tasks", task)
	defer resp.Body.Close()

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

func TestDBListCollections(t *testing.T) {
	req := httptest.NewRequest("GET", "/sudolistall", nil)
	w := httptest.NewRecorder()

	req.Header.Set("SB-PUBLIC-KEY", pubKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rootToken))

	stdRoot := []middleware.Middleware{
		middleware.WithDB(backend.DB, backend.Cache, getStripePortalURL),
		middleware.RequireRoot(backend.DB, backend.Cache),
	}
	h := middleware.Chain(http.HandlerFunc(db.listCollections), stdRoot...)

	h.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		t.Errorf("got error for list all collections: %s", string(b))
	}

	var names []string
	if err := parseBody(resp.Body, &names); err != nil {
		t.Fatal(err)
	} else if len(names) < 2 {
		t.Errorf("expected len to be > than 2 got %d", len(names))
	}
}

func TestListDocumentsInvalidDB(t *testing.T) {
	req := httptest.NewRequest("GET", "/db/invalid_db_name", nil)
	w := httptest.NewRecorder()

	req.Header.Set("SB-PUBLIC-KEY", pubKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rootToken))

	stdRoot := []middleware.Middleware{
		middleware.WithDB(backend.DB, backend.Cache, getStripePortalURL),
		middleware.RequireRoot(backend.DB, backend.Cache),
	}
	h := middleware.Chain(http.HandlerFunc(db.list), stdRoot...)

	h.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		t.Errorf("got error for list documents: %s", string(b))
	}
	expected := model.PagedResult{Page: 1, Size: 25}

	var response model.PagedResult
	if err := parseBody(resp.Body, &response); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(expected, response) {
		t.Errorf("incorrect response is received\nExpected: %#v\nActual: %#v", expected, response)
	}

}

func TestDBBulkUpdate(t *testing.T) {
	tasks := []Task{
		{
			Title: "should be updated",
			Done:  false,
		},
		{
			Title: "should be updated",
			Done:  false,
		},
	}

	resp := dbReq(t, db.bulkAdd, "POST", "/db/tasks/bulk", tasks)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var data = new(struct {
		UpdateFields map[string]any  `json:"update"`
		Clauses      [][]interface{} `json:"clauses"`
	})
	data.UpdateFields = map[string]any{"done": true}
	data.Clauses = append(data.Clauses, []interface{}{"title", "=", "should be updated"})

	resp = dbReq(t, db.bulkUpdate, "PUT", "/db/tasks/bulk", data)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var result int
	if err := parseBody(resp.Body, &result); err != nil {
		t.Fatal(err)
	} else if result != 2 {
		t.Errorf("expected count to be 2 got %d", result)
	}
}

func TestDBGetByIds(t *testing.T) {
	var data []string

	tasks := []Task{
		{
			Title: "should be returned 1",
			Done:  false,
		},
		{
			Title: "should be returned 2",
			Done:  false,
		},
	}

	var createdTasks []Task

	for _, v := range tasks {
		resp := dbReq(t, db.add, "POST", "/db/tasks", v)
		defer resp.Body.Close()

		if resp.StatusCode > 299 {
			t.Fatal(GetResponseBody(t, resp))
		}

		var saved Task
		if err := parseBody(resp.Body, &saved); err != nil {
			t.Fatal(err)
		}
		data = append(data, saved.ID)
		createdTasks = append(createdTasks, saved)
	}

	resp := dbReq(t, db.getByIds, "POST", "/db/tasks?byids=1", data)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var result []Task
	if err := parseBody(resp.Body, &result); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(createdTasks, result) {
		t.Errorf("Received incorrect data\nexpected: %v\ngot: %v", createdTasks, result)
	}
}

func TestDBIncrease(t *testing.T) {
	task :=
		Task{
			Title:   "item created",
			Created: time.Now(),
			Count:   1,
		}

	resp := dbReq(t, db.add, "POST", "/db/tasks", task)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var createdTask Task
	if err := parseBody(resp.Body, &createdTask); err != nil {
		t.Fatal(err)
	}

	var data = new(struct {
		Field string `json:"field"`
		Range int    `json:"range"`
	})
	data.Field = "count"
	data.Range = 4

	resp = dbReq(t, db.increase, "PUT", "/inc/tasks/"+createdTask.ID, data)

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	resp = dbReq(t, db.get, "GET", "/db/tasks/"+createdTask.ID, nil)
	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var increased Task
	if err := parseBody(resp.Body, &increased); err != nil {
		t.Fatal(err)
	} else if increased.Count != 5 {
		t.Errorf("expected count to be 5 got %d", increased.Count)
	}
}

func TestDBCreateIndex(t *testing.T) {
	req := httptest.NewRequest("POST", "/sudo/index?col=tasks&field=done", nil)
	w := httptest.NewRecorder()

	req.Header.Set("SB-PUBLIC-KEY", pubKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rootToken))

	stdRoot := []middleware.Middleware{
		middleware.WithDB(backend.DB, backend.Cache, getStripePortalURL),
		middleware.RequireRoot(backend.DB, backend.Cache),
	}
	h := middleware.Chain(http.HandlerFunc(db.index), stdRoot...)

	h.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		t.Errorf("got error for list all collections: %s", string(b))
	}

	//TODO: would be nice to validate the index were created
	// but there's no way to get a collection's indexes for now.
}
