package staticbackend

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	views map[string]*template.Template
)

func loadTemplates() error {
	var partials []string
	entries, err := os.ReadDir("./templates/partials")
	if err != nil {
		return err
	}

	funcs := customFuncs()

	for _, e := range entries {
		partials = append(partials, fmt.Sprintf("./templates/partials/%s", e.Name()))
	}

	views = make(map[string]*template.Template)

	tmpls, err := os.ReadDir("./templates")
	if err != nil {
		return err
	}

	for _, tmpl := range tmpls {
		name := fmt.Sprintf("./templates/%s", tmpl.Name())
		if !strings.HasSuffix(name, ".html") {
			continue
		}

		cur := append([]string{name}, partials...)

		t, err := template.New(tmpl.Name()).Funcs(funcs).ParseFiles(cur...)
		if err != nil {
			return err
		}

		views[tmpl.Name()] = t
	}

	return nil
}

type Flash struct {
	Type    string
	Message string
}

type ViewData struct {
	Title      string
	Language   string
	ActiveMenu string
	Flash      *Flash
	Data       interface{}
}

func render(w http.ResponseWriter, r *http.Request, view string, data interface{}, flash *Flash) {
	renderWithMenu(w, r, view, data, flash, "")
}

func renderWithMenu(w http.ResponseWriter, r *http.Request, view string, data interface{}, flash *Flash, menu string) {
	vd := ViewData{
		ActiveMenu: menu,
		Data:       data,
		Flash:      flash,
	}

	tmpl, ok := views[view]
	if !ok {
		http.Error(w, fmt.Sprintf(`template "%s" cannot be found`, view), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, vd); err != nil {
		//TODO: log this, it's important
		log.Printf(`error executing template "%s" got %v`, view, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func renderErr(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		//TODO: log this
		log.Println("err in ui", err)
	}
	render(w, r, "err.html", nil, nil)
}

func customFuncs() template.FuncMap {
	return template.FuncMap{
		"getField": func(s string, doc interface{}) string {
			dict, ok := doc.(bson.M)
			if !ok {
				return "error converting document"
			}
			v, ok := dict[s]
			if !ok {
				return "n/a"
			}

			oid, ok := v.(primitive.ObjectID)
			if ok {
				return oid.Hex()
			}

			date, ok := dict[s].(time.Time)
			if ok {
				return date.Format("2006-01-02 15:04")
			}

			if s == "sb_posted" {
				i, err := strconv.ParseInt(fmt.Sprintf("%v", dict[s]), 10, 64)
				if err == nil {
					ts := time.Unix(i/1000, 0)
					return ts.Format("2006-01-02 15:04")
				}
			}
			return fmt.Sprintf("%v", v)
		},
	}
}
