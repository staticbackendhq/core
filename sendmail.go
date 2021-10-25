package staticbackend

import (
	"context"
	"net/http"
	"staticbackend/email"
	"staticbackend/internal"
	"staticbackend/middleware"

	"go.mongodb.org/mongo-driver/bson"
)

func sudoSendMail(w http.ResponseWriter, r *http.Request) {
	var data internal.SendMailData
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// if only body is provided
	if len(data.Body) > 0 {
		data.HTMLBody = data.Body
		data.TextBody = email.StripHTML(data.Body)
	} else if len(data.TextBody) == 0 && len(data.HTMLBody) > 0 {
		data.TextBody = email.StripHTML(data.HTMLBody)
	} else if len(data.HTMLBody) == 0 && len(data.TextBody) > 0 {
		data.HTMLBody = data.TextBody
	}

	if err := emailer.Send(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := client.Database("sbsys")

	ctx := context.Background()

	filter := bson.M{internal.FieldID: config.SBID}
	update := bson.M{"$inc": bson.M{"mes": 1}}
	if _, err := db.Collection("accounts").UpdateOne(ctx, filter, update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}
