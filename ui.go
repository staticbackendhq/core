package staticbackend

import (
	"net/http"
	"staticbackend/db"
	"staticbackend/internal"
	"staticbackend/middleware"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ui struct {
	base *db.Base
}

func (ui) login(w http.ResponseWriter, r *http.Request) {
	render(w, r, "login.html", nil, nil)
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
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		renderErr(w, r, err)
		return
	}

	curDB := client.Database(conf.Name)

	names, err := x.base.ListCollections(curDB)
	if err != nil {
		renderErr(w, r, err)
		return
	}

	render(w, r, "db_cols.html", names, nil)
}
