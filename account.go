package main

import (
	"context"
	"net/http"
	"time"

	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/billingportal/session"
	"github.com/stripe/stripe-go/v71/customer"
	"github.com/stripe/stripe-go/v71/sub"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	base := BaseConfig{
		ID:        primitive.NewObjectID(),
		SBID:      acctID,
		Name:      email,
		Whitelist: []string{"localhost"},
	}

	if _, err := db.Collection("bases").InsertOne(ctx, base); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//TODO: send email with their new base key.

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(cus.ID),
		ReturnURL: stripe.String("https://staticbackend.com/stripe"),
	}
	s, err := session.New(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, s.URL)

}
