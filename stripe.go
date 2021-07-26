package staticbackend

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"staticbackend/internal"

	"go.mongodb.org/mongo-driver/bson"
)

type stripeWebhook struct{}

type CustomerSourceCreated struct {
	Data struct {
		CustomerID string `json:"customer"`
	} `json:"data"`
}

func (wh *stripeWebhook) process(w http.ResponseWriter, r *http.Request) {
	fmt.Println("inside stripe webhook")
	var evt map[string]interface{}
	if err := parseBody(r.Body, &evt); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	typ, ok := evt["type"]
	if !ok {
		log.Println("no type specified")
		http.Error(w, "no type specified in the event data", http.StatusBadRequest)
		return
	}

	fmt.Println("event type", typ)

	var err error

	switch typ {
	case "customer.source.created":
		err = wh.sourceCreated(evt["data"])
	}

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func (wh *stripeWebhook) sourceCreated(params interface{}) error {
	data, ok := params.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unable to cast params: %v into a map[string]interface{}", params)
	}

	obj, ok := data["object"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unable to cast data[object] into a map[string]interface{}")
	}

	stripeID, ok := obj["customer"].(string)
	if !ok {
		return fmt.Errorf("unable to convert %v to string", data["customer"])
	}

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
