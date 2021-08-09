package staticbackend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"staticbackend/db"
	"staticbackend/internal"
	"staticbackend/middleware"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ui struct {
	base *db.Base
}

func (ui) login(w http.ResponseWriter, r *http.Request) {
	render(w, r, "login.html", nil, nil)
}

func (ui) createApp(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	email := strings.ToLower(r.URL.Query().Get("email"))
	// TODO: cheap email validation
	if len(email) < 4 || strings.Index(email, "@") == -1 || strings.Index(email, ".") == -1 {
		http.Error(w, "invalid email", http.StatusBadRequest)
		return
	}

	db := client.Database("sbsys")
	exists, err := internal.EmailExists(db, email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if exists {
		http.Error(w, "Please use a different/valid email.", http.StatusInternalServerError)
		return
	}

}

func (ui) auth(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	pk := r.Form.Get("pk")
	token := r.Form.Get("token")

	id, err := primitive.ObjectIDFromHex(pk)
	if err != nil {
		render(w, r, "login.html", nil, &Flash{Type: "danger", Message: "This app does not exists"})
		return
	}

	db := client.Database("sbsys")

	conf, err := internal.FindDatabase(db, id)
	if err != nil {
		render(w, r, "login.html", nil, &Flash{Type: "danger", Message: "This app does not exists"})
		return
	}

	if _, err := middleware.ValidateRootToken(client, conf.Name, token); err != nil {
		render(w, r, "login.html", nil, &Flash{Type: "danger", Message: "invalid public key / token"})
		return
	}

	ckToken := &http.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Now().Add(2 * 24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	}
	http.SetCookie(w, ckToken)

	ckPk := &http.Cookie{
		Name:     "pk",
		Value:    pk,
		Expires:  time.Now().Add(2 * 24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	}
	http.SetCookie(w, ckPk)

	http.Redirect(w, r, "/ui/db", http.StatusSeeOther)
}

func (x *ui) dbCols(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, false)
	if err != nil {
		renderErr(w, r, err)
		return
	}

	curDB := client.Database(conf.Name)

	data := new(struct {
		Collection     string
		Collections    []string
		Columns        []string
		Docs           []bson.M
		SortBy         string
		SortDescending string
		FilterFields   string
		Query          string
	})

	names, err := x.base.ListCollections(curDB)
	if err != nil {
		renderErr(w, r, err)
		return
	}

	if len(names) == 0 {
		render(w, r, "db_cols.html", names, nil)
	}

	col := names[0]

	params := db.ListParams{
		Page:           1,
		Size:           50,
		SortDescending: true,
		SortBy:         "id",
	}

	var filter bson.M

	// handle post
	if r.Method == http.MethodPost {
		r.ParseForm()

		col = r.Form.Get("col")
		params.SortBy = r.Form.Get("sortby")
		params.SortDescending = r.Form.Get("desc") == "1"

		query := r.Form.Get("query")
		if len(query) > 0 {
			data.Query = query

			var clauses [][]interface{}
			if err := json.Unmarshal([]byte(query), &clauses); err != nil {
				renderErr(w, r, err)
				return
			}

			filter, err = db.ParseQuery(clauses)
			if err != nil {
				renderErr(w, r, err)
				return
			}
		}
	}

	var list db.PagedResult
	if len(filter) == 0 {
		list, err = x.base.List(auth, curDB, col, params)
		if err != nil {
			renderErr(w, r, err)
			return
		}
	} else {
		list, err = x.base.Query(auth, curDB, col, filter, params)
		if err != nil {
			renderErr(w, r, err)
			return
		}
	}

	columns := x.readColumnNames(list.Results)

	data.Collection = col
	data.Collections = names
	data.Columns = columns
	data.Docs = list.Results
	data.SortBy = params.SortBy
	if params.SortDescending {
		data.SortDescending = "1"
	} else {
		data.SortDescending = "0"
	}

	render(w, r, "db_cols.html", data, nil)
}

func (x ui) dbDoc(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		renderErr(w, r, err)
		return
	}

	curDB := client.Database(conf.Name)

	col := r.URL.Query().Get("col")
	id := getURLPart(r.URL.Path, 3)

	doc, err := x.base.GetByID(auth, curDB, col, id)
	if err != nil {
		renderErr(w, r, err)
		return
	}

	data := new(struct {
		Collection string
		Columns    []string
		Doc        interface{}
	})

	var docs []bson.M
	docs = append(docs, doc)

	data.Collection = col
	data.Columns = x.readColumnNames(docs)
	data.Doc = doc

	render(w, r, "db_doc.html", data, nil)
}

func (x ui) dbSave(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		renderErr(w, r, err)
		return
	}

	curDB := client.Database(conf.Name)

	id := r.Form.Get("id")
	col := r.Form.Get("col")
	field := r.Form.Get("field")
	value := r.Form.Get("value")
	typ := r.Form.Get("type")

	update := bson.M{}

	if typ == "int" {
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			renderErr(w, r, err)
			return
		}

		update[field] = i
	} else if typ == "float" {
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			renderErr(w, r, err)
			return
		}

		update[field] = f
	} else if typ == "bool" {
		update[field] = value == "true"
	} else {
		update[field] = value
	}

	if _, err := x.base.Update(auth, curDB, col, id, update); err != nil {
		renderErr(w, r, err)
		return
	}

	url := fmt.Sprintf("/ui/db/%s?col=%s", id, col)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (x ui) dbDel(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		renderErr(w, r, err)
		return
	}

	curDB := client.Database(conf.Name)

	col := r.URL.Query().Get("col")
	id := getURLPart(r.URL.Path, 4)

	if _, err := x.base.Delete(auth, curDB, col, id); err != nil {
		renderErr(w, r, err)
		return
	}

	http.Redirect(w, r, "/ui/db", http.StatusSeeOther)
}

func (ui) readColumnNames(docs []bson.M) []string {
	if len(docs) == 0 {
		return nil
	}

	first := docs[0]

	var columns []string
	columns = append(columns, "id")
	columns = append(columns, "accountId")

	for k, _ := range first {
		if strings.EqualFold(k, "id") {
			continue
		} else if strings.EqualFold(k, "accountId") {
			continue
		}

		columns = append(columns, k)
	}

	return columns
}

func (x ui) forms(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		renderErr(w, r, err)
		return
	}

	formName := r.URL.Query().Get("fn")

	curDB := client.Database(conf.Name)

	forms, err := internal.GetForms(curDB)
	if err != nil {
		renderErr(w, r, err)
		return
	}

	entries, err := internal.ListFormSubmissions(curDB, formName)
	if err != nil {
		renderErr(w, r, err)
		return
	}

	var data = new(struct {
		FormName string
		Forms    []string
		Entries  []bson.M
	})

	data.FormName = formName
	data.Forms = forms
	data.Entries = entries

	render(w, r, "forms.html", data, nil)
}

func (x ui) formDel(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		renderErr(w, r, err)
		return
	}

	curDB := client.Database(conf.Name)

	id := getURLPart(r.URL.Path, 4)

	if _, err := x.base.Delete(auth, curDB, "sb_forms", id); err != nil {
		renderErr(w, r, err)
		return
	}

	http.Redirect(w, r, "/ui/forms", http.StatusSeeOther)
}
