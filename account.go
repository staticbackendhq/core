package staticbackend

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	emailFuncs "github.com/staticbackendhq/core/email"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/middleware"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/billingportal/session"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/sub"
)

var (
	FromEmail   = os.Getenv("FROM_EMAIL")
	FromName    = os.Getenv("FROM_NAME")
	letterRunes = []rune("abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ2345679")
)

type accounts struct {
	membership *membership
}

func (a *accounts) create(w http.ResponseWriter, r *http.Request) {
	var email string
	fromCLI := true

	// the CLI do a GET for the account initialization, we can then
	// base the rest of the flow on the fact that the web UI POST data
	if r.Method == http.MethodPost {
		fromCLI = false

		r.ParseForm()

		email = strings.ToLower(r.Form.Get("email"))
	} else {
		email = strings.ToLower(r.URL.Query().Get("email"))
	}
	// TODO: cheap email validation
	if len(email) < 4 || strings.Index(email, "@") == -1 || strings.Index(email, ".") == -1 {
		http.Error(w, "invalid email", http.StatusBadRequest)
		return
	}

	exists, err := datastore.EmailExists(email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if exists {
		http.Error(w, "Please use a different/valid email.", http.StatusInternalServerError)
		return
	}

	stripeCustomerID, subID := "", ""
	active := true

	if AppEnv == AppEnvProd && len(os.Getenv("STRIPE_KEY")) > 0 {
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
					Price: stripe.String(os.Getenv("STRIPE_PRICEID_IDEA")),
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

	cust := internal.Customer{
		Email:          email,
		StripeID:       stripeCustomerID,
		SubscriptionID: subID,
		Plan:           internal.PlanIdea,
		IsActive:       active,
		Created:        time.Now(),
	}

	cust, err = datastore.CreateCustomer(cust)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// make sure the DB name is unique
	retry := 10
	dbName := randStringRunes(12)
	for {
		exists, err = datastore.DatabaseExists(dbName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if exists {
			retry--
			dbName = randStringRunes(12)
			continue
		}
		break
	}

	base := internal.BaseConfig{
		CustomerID:    cust.ID,
		Name:          dbName,
		IsActive:      active,
		AllowedDomain: []string{"localhost"},
	}

	bc, err := datastore.CreateBase(base)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// we create an admin user
	// we make sure to switch DB
	pw := randStringRunes(6)

	if _, _, err := a.membership.createAccountAndUser(dbName, email, pw, 100); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	signUpURL := "no need to sign up in dev mode"
	if AppEnv == AppEnvProd && len(os.Getenv("STRIPE_KEY")) > 0 {
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

	token, err := datastore.FindTokenByEmail(dbName, email)
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

	ed := internal.SendMailData{
		From:     FromEmail,
		FromName: FromName,
		To:       email,
		ToName:   "",
		Subject:  "Your StaticBackend account",
		HTMLBody: body,
		TextBody: emailFuncs.StripHTML(body),
	}

	err = emailer.Send(ed)
	if err != nil {
		log.Println("error sending email", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if fromCLI {
		respond(w, http.StatusOK, signUpURL)
		return
	}

	if strings.HasPrefix(signUpURL, "https") {
		http.Redirect(w, r, signUpURL, http.StatusSeeOther)
		return
	}

	render(w, r, "login.html", nil, &Flash{Type: "sucess", Message: "We've emailed you all the information you need to get started."})
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

	cus, err := datastore.FindAccount(conf.CustomerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(cus.StripeID),
		ReturnURL: stripe.String("https://staticbackend.com/stripe"),
	}
	s, err := session.New(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, s.URL)
}

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	// due to PostgreSQL schema requiring letter start.
	b[0] = letterRunes[0]

	return string(b)
}
