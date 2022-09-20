package staticbackend

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/config"
	emailFuncs "github.com/staticbackendhq/core/email"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/model"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/billingportal/session"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/sub"
)

type accounts struct {
	log *logger.Logger
}

func (a *accounts) create(w http.ResponseWriter, r *http.Request) {
	var email string
	fromCLI := true
	memoryMode := false

	// the CLI do a GET for the account initialization, we can then
	// base the rest of the flow on the fact that the web UI POST data
	if r.Method == http.MethodPost {
		fromCLI = false

		r.ParseForm()

		email = strings.ToLower(r.Form.Get("email"))
	} else {
		email = strings.ToLower(r.URL.Query().Get("email"))

		if config.Current.AppEnv != AppEnvProd {
			memoryMode = r.URL.Query().Get("mem") == "1"
		}

		// the marketing website uses a query string ?ui=true
		if len(r.URL.Query().Get("ui")) > 0 {
			fromCLI = false
		}
	}
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

	stripeCustomerID, subID := "", ""
	active := true

	if config.Current.AppEnv == AppEnvProd && len(config.Current.StripeKey) > 0 {
		active = false

		cusParams := &stripe.CustomerParams{
			Email: stripe.String(email),
		}
		cus, err := customer.New(cusParams)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		stripeCustomerID = cus.ID

		subParams := &stripe.SubscriptionParams{
			Customer: stripe.String(cus.ID),
			Items: []*stripe.SubscriptionItemsParams{
				{
					Price: stripe.String(config.Current.StripePriceIDIdea),
				},
			},
			TrialPeriodDays: stripe.Int64(60),
		}
		newSub, err := sub.New(subParams)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		subID = newSub.ID
	}

	// create the account

	cust := model.Tenant{
		ID:             "cust-local-dev", // easier for CLI/memory flow
		Email:          email,
		StripeID:       stripeCustomerID,
		SubscriptionID: subID,
		Plan:           model.PlanIdea,
		IsActive:       active,
		Created:        time.Now(),
	}

	cust, err = backend.DB.CreateTenant(cust)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// make sure the DB name is unique
	retry := 10
	dbName := internal.RandStringRunes(12)
	if memoryMode {
		dbName = "dev-memory-pk"
	}
	for {
		exists, err = backend.DB.DatabaseExists(dbName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if exists {
			retry--
			dbName = internal.RandStringRunes(12)
			continue
		}
		break
	}

	base := model.DatabaseConfig{
		ID:            dbName, // easier for CLI/memory flow
		TenantID:      cust.ID,
		Name:          dbName,
		IsActive:      active,
		AllowedDomain: []string{"localhost"},
	}

	bc, err := backend.DB.CreateDatabase(base)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// we create an admin user
	// we make sure to switch DB
	pw := internal.RandStringRunes(6)
	if memoryMode {
		pw = "devpw1234"
	}

	mship := backend.Membership(base)
	if _, _, err := mship.CreateAccountAndUser(email, pw, 100); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	signUpURL := "no need to sign up in dev mode"
	if config.Current.AppEnv == AppEnvProd && len(config.Current.StripeKey) > 0 {
		params := &stripe.BillingPortalSessionParams{
			Customer:  stripe.String(stripeCustomerID),
			ReturnURL: stripe.String("https://staticbackend.com/stripe"),
		}
		s, err := session.New(params)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		signUpURL = s.URL
	}

	token, err := backend.DB.FindUserByEmail(dbName, email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rootToken := fmt.Sprintf("%s|%s|%s", token.ID, token.AccountID, token.Token)

	//TODO: Have html template for those
	body := fmt.Sprintf(`
	<p>Hey there,</p>
	<p>Thanks for creating your account.</p>
	<p>Your SB-PUBLIC-KEY is required on all your API requests:</p>
	<p>SB-PUBLIC-KEY: <strong>%s</strong></p>
	<p>We've created an admin user for your new database:</p>
	<p>email: %s<br />
	password: %s</p>
	<p>This is your root token key. You'll need this to manage your database and 
	execute "sudo" commands from your backend functions</p>
	<p>ROOT TOKEN: <strong>%s</strong></p>
	<p>Make sure you complete your account creation by entering a valid credit 
	card via the link you got when issuing the account create command.</p>
	<p>If you have any questions, please reply to this email.</p>
	<p>Good luck with your projects.</p>
	<p>Dominic<br />Founder</p>
	`, bc.ID, email, pw, rootToken)

	ed := emailFuncs.SendMailData{
		From:     config.Current.FromEmail,
		FromName: config.Current.FromName,
		To:       email,
		ToName:   "",
		Subject:  "Your StaticBackend account",
		HTMLBody: body,
		TextBody: emailFuncs.StripHTML(body),
	}

	if memoryMode {
		fmt.Printf(`
Start sending requests with the following credentials:


Public key:		%s


Admin user:
	Email:		%s
	Password:	%s


Dev root token:		safe-to-use-in-dev-root-token
Real root token:		%s


Refer to the documentation at https://staticbackend.com/docs\n

`,
			bc.ID, email, pw, rootToken,
		)

		// cache the root token so caller can always use
		// "safe-to-use-in-dev-root-token" as root token instead of
		// the changing one across CLI start/stop
		backend.Cache.Set("dev-root-token", rootToken)
	} else {
		err = backend.Emailer.Send(ed)
		if err != nil {
			a.log.Error().Err(err).Msg("error sending email")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	a.log.Debug().Msgf("DEBUG: %s", signUpURL)

	if fromCLI {
		respond(w, http.StatusOK, signUpURL)
		return
	}

	if strings.HasPrefix(signUpURL, "https") {
		http.Redirect(w, r, signUpURL, http.StatusSeeOther)
		return
	}

	render(w, r, "login.html", nil, &Flash{Type: "sucess", Message: "We've emailed you all the information you need to get started."}, a.log)
}

func (a *accounts) auth(w http.ResponseWriter, r *http.Request) {
	_, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	respond(w, http.StatusOK, auth.Email)
}

func (a *accounts) portal(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	url, err := getStripePortalURL(conf.TenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, url)
}

func getStripePortalURL(customerID string) (string, error) {
	cus, err := backend.DB.FindTenant(customerID)
	if err != nil {
		return "", err
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(cus.StripeID),
		ReturnURL: stripe.String("https://staticbackend.com/stripe"),
	}
	s, err := session.New(params)
	if err != nil {
		return "", err
	}

	return s.URL, nil
}
