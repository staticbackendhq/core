package staticbackend

import (
	"log"
	"net/http"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/email"
	"github.com/staticbackendhq/core/middleware"
)

func sudoSendMail(w http.ResponseWriter, r *http.Request) {
	var data email.SendMailData
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

	if err := backend.Emailer.Send(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := backend.DB.IncrementMonthlyEmailSent(config.ID); err != nil {
		//TODO: do something better with this error
		log.Println("error increasing monthly email sent: ", err)
	}

	respond(w, http.StatusOK, true)
}
