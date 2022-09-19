package staticbackend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/model"
)

type ui struct {
	log *logger.Logger
}

func (ui) login(w http.ResponseWriter, r *http.Request) {
	render(w, r, "login.html", nil, nil, nil)
}

func (ui) createApp(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	email := strings.ToLower(r.URL.Query().Get("email"))
	// TODO: cheap email validation
	if len(email) < 4 || strings.Index(email, "@") == -1 || strings.Index(email, ".") == -1 {
		http.Error(w, "invalid email", http.StatusBadRequest)
		return
	}

	exists, err := backend.DB.EmailExists(email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if exists {
		http.Error(w, "Please use a different/valid email.", http.StatusInternalServerError)
		return
	}

}

func (x ui) auth(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	pk := r.Form.Get("pk")
	token := r.Form.Get("token")

	conf, err := backend.DB.FindDatabase(pk)
	if err != nil {
		render(w, r, "login.html", nil, &Flash{Type: "danger", Message: "This app does not exists"}, x.log)
		return
	}

	if _, err := middleware.ValidateRootToken(backend.DB, conf.Name, token); err != nil {
		render(w, r, "login.html", nil, &Flash{Type: "danger", Message: "invalid public key / token"}, x.log)
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

func (x ui) logins(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	cus, err := backend.DB.FindTenant(conf.TenantID)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	logins, err := cus.GetExternalLogins()
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	render(w, r, "logins.html", logins, nil, x.log)
}

func (x ui) enableExternalLogin(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	cus, err := backend.DB.FindTenant(conf.TenantID)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	r.ParseForm()
	provider := r.Form.Get("provider")
	apikey := r.Form.Get("apikey")
	secret := r.Form.Get("apisecret")

	logins, err := cus.GetExternalLogins()
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	keys, ok := logins[provider]
	if !ok {
		keys = model.OAuthConfig{}
	}

	keys.ConsumerKey = apikey
	keys.ConsumerSecret = secret

	logins[provider] = keys

	if err := backend.DB.EnableExternalLogin(cus.ID, logins); err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	flash := &Flash{Type: "success", Message: "OAuth provider successfully added"}
	render(w, r, "logins.html", logins, flash, x.log)
}

func (x *ui) dbCols(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, false)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	data := new(struct {
		Collection     string
		Collections    []string
		Columns        []string
		Docs           []map[string]interface{}
		SortBy         string
		SortDescending string
		FilterFields   string
		Query          string
	})

	allNames, err := backend.DB.ListCollections(conf.Name)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	// we remove the "system" collection
	var names []string
	for _, name := range allNames {
		if strings.HasPrefix(name, "sb_") {
			continue
		}

		names = append(names, name)
	}

	if len(names) == 0 {
		render(w, r, "db_cols.html", data, nil, x.log)
		return
	}

	col := names[0]

	params := model.ListParams{
		Page:           1,
		Size:           50,
		SortDescending: true,
		SortBy:         "id",
	}

	filter := make(map[string]interface{})

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
				renderErr(w, r, err, x.log)
				return
			}

			filter, err = backend.DB.ParseQuery(clauses)
			if err != nil {
				renderErr(w, r, err, x.log)
				return
			}
		}
	}

	var list model.PagedResult
	if !strings.HasPrefix(col, "sb_") {
		if len(filter) == 0 {
			list, err = backend.DB.ListDocuments(auth, conf.Name, col, params)
			if err != nil {
				renderErr(w, r, err, x.log)
				return
			}
		} else {
			list, err = backend.DB.QueryDocuments(auth, conf.Name, col, filter, params)
			if err != nil {
				renderErr(w, r, err, x.log)
				return
			}
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

	render(w, r, "db_cols.html", data, nil, x.log)
}

func (x ui) dbDoc(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	col := r.URL.Query().Get("col")
	id := getURLPart(r.URL.Path, 3)

	doc, err := backend.DB.GetDocumentByID(auth, conf.Name, col, id)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	data := new(struct {
		Collection string
		Columns    []string
		Doc        interface{}
	})

	var docs []map[string]interface{}
	docs = append(docs, doc)

	data.Collection = col
	data.Columns = x.readColumnNames(docs)
	data.Doc = doc

	render(w, r, "db_doc.html", data, nil, x.log)
}

func (x ui) dbSave(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	id := r.Form.Get("id")
	col := r.Form.Get("col")
	field := r.Form.Get("field")
	value := r.Form.Get("value")
	typ := r.Form.Get("type")

	update := make(map[string]interface{})

	if typ == "int" {
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			renderErr(w, r, err, x.log)
			return
		}

		update[field] = i
	} else if typ == "float" {
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			renderErr(w, r, err, x.log)
			return
		}

		update[field] = f
	} else if typ == "bool" {
		update[field] = value == "true"
	} else {
		update[field] = value
	}

	if _, err := backend.DB.UpdateDocument(auth, conf.Name, col, id, update); err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	url := fmt.Sprintf("/ui/db/%s?col=%s", id, col)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (x ui) dbDel(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	col := r.URL.Query().Get("col")
	id := getURLPart(r.URL.Path, 4)

	if _, err := backend.DB.DeleteDocument(auth, conf.Name, col, id); err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	http.Redirect(w, r, "/ui/db", http.StatusSeeOther)
}

func (ui) readColumnNames(docs []map[string]interface{}) []string {
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
		renderErr(w, r, err, x.log)
		return
	}

	formName := r.URL.Query().Get("fn")

	forms, err := backend.DB.GetForms(conf.Name)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	entries, err := backend.DB.ListFormSubmissions(conf.Name, formName)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	var data = new(struct {
		FormName string
		Forms    []string
		Entries  []map[string]interface{}
	})

	data.FormName = formName
	data.Forms = forms
	data.Entries = entries

	render(w, r, "forms.html", data, nil, x.log)
}

func (x ui) formDel(w http.ResponseWriter, r *http.Request) {
	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	id := getURLPart(r.URL.Path, 4)

	if _, err := backend.DB.DeleteDocument(auth, conf.Name, "sb_forms", id); err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	http.Redirect(w, r, "/ui/forms", http.StatusSeeOther)
}

func (x ui) fnList(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	results, err := backend.DB.ListFunctions(conf.Name)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	render(w, r, "fn_list.html", results, nil, x.log)
}

func (x *ui) fnNew(w http.ResponseWriter, r *http.Request) {
	fn := model.ExecData{}
	render(w, r, "fn_edit.html", fn, nil, x.log)
}

func (x *ui) fnEdit(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	id := getURLPart(r.URL.Path, 3)

	fn, err := backend.DB.GetFunctionByID(conf.Name, id)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	render(w, r, "fn_edit.html", fn, nil, x.log)
}

func (x *ui) fnSave(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	r.ParseForm()

	id := r.Form.Get("id")
	name := r.Form.Get("name")
	trigger := r.Form.Get("trigger")
	code := r.Form.Get("code")

	if id == "new" {
		fn := model.ExecData{
			FunctionName: name,
			Code:         code,
			TriggerTopic: trigger,
		}
		newID, err := backend.DB.AddFunction(conf.Name, fn)
		if err != nil {
			renderErr(w, r, err, x.log)
			return
		}

		http.Redirect(w, r, "/ui/fn/"+newID, http.StatusSeeOther)
		return
	}

	if err := backend.DB.UpdateFunction(conf.Name, id, code, trigger); err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	http.Redirect(w, r, "/ui/fn/"+id, http.StatusSeeOther)
}

func (x *ui) fnDel(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}
	name := getURLPart(r.URL.Path, 4)
	if err := backend.DB.DeleteFunction(conf.Name, name); err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	http.Redirect(w, r, "/ui/fn", http.StatusSeeOther)
}

func (x *ui) fsList(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	accountID := r.URL.Query().Get("id")

	results, err := backend.DB.ListAllFiles(conf.Name, accountID)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	render(w, r, "fs_list.html", results, nil, x.log)
}

func (x *ui) fsDel(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	fileID := getURLPart(r.URL.Path, 4)

	if err := backend.DB.DeleteFile(conf.Name, fileID); err != nil {
		renderErr(w, r, err, x.log)
		return
	}

	http.Redirect(w, r, "/ui/fs", http.StatusSeeOther)
}
