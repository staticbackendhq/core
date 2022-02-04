package staticbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/staticbackendhq/core/internal"

	"github.com/stripe/stripe-go/v71"
	"go.mongodb.org/mongo-driver/bson"
)

type stripeWebhook struct{}

func (wh *stripeWebhook) process(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	event := stripe.Event{}

	if err := json.Unmarshal(payload, &event); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse webhook body json: %v\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Unmarshal the event data into an appropriate struct depending on its Type
	switch event.Type {
	case "payment_method.attached":
		var paymentMethod stripe.PaymentMethod
		err := json.Unmarshal(event.Data.Raw, &paymentMethod)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		wh.handlePaymentMethodAttached(paymentMethod)
	}

	w.WriteHeader(http.StatusOK)
}

func (wh *stripeWebhook) handlePaymentMethodAttached(pm stripe.PaymentMethod) error {
	stripeID := pm.Customer.ID

	db := client.Database("sbsys")
	ctx := context.Background()

	var acct internal.Customer
	sr := db.Collection("accounts").FindOne(ctx, bson.M{"stripeId": stripeID})
	if err := sr.Decode(&acct); err != nil {
		return err
	} else if err := sr.Err(); err != nil {
		fmt.Println("customer", stripeID)
		return err
	}

	if acct.IsActive {
		return nil
	}

	filter := bson.M{"_id": acct.ID}
	update := bson.M{"$set": bson.M{"active": true}}

	res := db.Collection("accounts").FindOneAndUpdate(ctx, filter, update)
	if err := res.Err(); err != nil {
		fmt.Printf("error while saving account: %v %s %s\n", err, stripeID, acct.ID.Hex())
		return err
	}

	filter = bson.M{internal.FieldAccountID: acct.ID}
	res = db.Collection("bases").FindOneAndUpdate(ctx, filter, update)
	return res.Err()
}
