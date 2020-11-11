package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/billingportal/session"
	"github.com/stripe/stripe-go/v71/customer"
	"github.com/stripe/stripe-go/v71/sub"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	FromEmail = "support@staticbackend.com"
	FromName  = "StaticBackend Support"
)

var (
	letterRunes = []rune("abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ2345679")
)

type accounts struct{}

type Customer struct {
	ID             primitive.ObjectID `bson:"_id" json:"id"`
	Email          string             `bson:"email" json:"email"`
	StripeID       string             `bson:"stripeId" json:"stripeId"`
	SubscriptionID string             `bson:"subId" json:"subId"`
	IsActive       bool               `bson:"active" json:"-"`
	Created        time.Time          `bson:"created" json:"created"`
}

func (a *accounts) create(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")

	db := client.Database("sbsys")
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	count, err := db.Collection("accounts").CountDocuments(ctx, bson.M{"email": email})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if count > 0 {
		http.Error(w, "Please use a different/valid email.", http.StatusInternalServerError)
		return
	}

	cusParams := &stripe.CustomerParams{
		Email: stripe.String(email),
	}
	cus, err := customer.New(cusParams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	subParams := &stripe.SubscriptionParams{
		Customer: stripe.String(cus.ID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String("price_1HExopLi4uPpotEYCGblikpg"),
			},
		},
		TrialPeriodDays: stripe.Int64(14),
	}
	newSub, err := sub.New(subParams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// create the account
	acctID := primitive.NewObjectID()
	doc := Customer{
		ID:             acctID,
		Email:          email,
		StripeID:       cus.ID,
		SubscriptionID: newSub.ID,
		Created:        time.Now(),
	}

	if _, err := db.Collection("accounts").InsertOne(ctx, doc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// make sure the DB name is unique
	retry := 10
	dbName := randStringRunes(12)
	for {
		count, err = db.Collection("bases").CountDocuments(ctx, bson.M{"name": dbName})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if count > 0 {
			retry--
			dbName = randStringRunes(12)
			continue
		}
		break
	}

	base := BaseConfig{
		ID:        primitive.NewObjectID(),
		SBID:      acctID,
		Name:      dbName,
		Whitelist: []string{"localhost"},
	}

	if _, err := db.Collection("bases").InsertOne(ctx, base); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// we create an admin user
	// we make sure to switch DB
	db = client.Database(dbName)
	pw := randStringRunes(6)

	if _, err := createAccountAndUser(db, email, pw, 100); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(cus.ID),
		ReturnURL: stripe.String("https://staticbackend.com/stripe"),
	}
	s, err := session.New(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body := fmt.Sprintf(`
	<p>Hey there,</p>
	<p>Thanks for creating your account.</p>
	<p>Your SB-PUBLIC-KEY is required on all your API requests:</p>
	<p>SB-PUBLUC-KEY: <strong>%s</strong></p>
	<p>We've created an admin user for your new database:</p>
	<p>email: %s<br />
	password: %s</p>
	<p>Make sure you complete your account creation by entering a valid credit 
	card via the email you got when issuing the account create command.</p>
	<p>If you have any questions, please reply to this email.</p>
	<p>Good luck with your projects.</p>
	<p>Dominic<br />Founder</p>
	`, base.ID.Hex(), email, pw)

	fmt.Println("sending email to new user", email)
	err = sendMail(email, "", FromEmail, FromName, "Your StaticBackend account", body, "")
	if err != nil {
		log.Println("error sending email", err)
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

	return string(b)
}
