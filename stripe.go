package staticbackend

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/stripe/stripe-go/v71"
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

	acct, err := datastore.GetCustomerByStripeID(stripeID)
	if err != nil {
		return err
	}

	if acct.IsActive {
		return nil
	}

	return datastore.ActivateCustomer(acct.ID)
}
