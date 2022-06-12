package staticbackend

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/internal"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/webhook"
)

// TODO: Implement better logging mechanism than std output

type stripeWebhook struct{}

func (wh *stripeWebhook) process(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("STRIPE ERROR (read body): ", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	endpointSecret := config.Current.StripeWebhookSecret

	// Verify webhook signature and extract the event.
	// See https://stripe.com/docs/webhooks/signatures for more information.
	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), endpointSecret)
	if err != nil {
		fmt.Println("STRIPE ERROR (verify secret): ", err)
		w.WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature.
		return
	}

	if event.Type == "customer.subscription.updated" {
		var sub stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &sub)
		if err != nil {
			fmt.Println("STRIPE ERROR (sub update json)): ", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		go wh.handleSubChanged(sub)
	} else if event.Type == "customer.subscription.deleted" {
		var sub stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &sub)
		if err != nil {
			fmt.Println("STRIPE ERROR (sub del json)): ", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		go wh.handleSubCancelled(sub)
	} else if event.Type == "payment_method.attached" {
		var paymentMethod stripe.PaymentMethod
		err := json.Unmarshal(event.Data.Raw, &paymentMethod)
		if err != nil {
			fmt.Fprintf(os.Stderr, "STRIPE ERROR: parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		go wh.handlePaymentMethodAttached(paymentMethod)
	} else {
		log.Printf("received unhandled Stripe webhook: %s\n", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

func (wh *stripeWebhook) handleSubChanged(sub stripe.Subscription) {
	stripeID := sub.Customer.ID

	fmt.Println("[Sub Changed]: for StripeID: ", stripeID)

	// find the customer
	cus, err := datastore.GetCustomerByStripeID(stripeID)
	if err != nil {
		fmt.Println("STRIPE ERROR (find cus by stripe id): ", err)
		return
	}

	fmt.Println("[Sub Changed]: found account: ", cus.Email)

	if sub.Items.TotalCount > 0 {
		fmt.Println("[Sub Changed]: there's at least 1 sub")

		priceID := sub.Items.Data[0].Price.ID
		newLevel := wh.priceToLevel(priceID)

		if err := datastore.ChangeCustomerPlan(cus.ID, newLevel); err != nil {
			fmt.Println("STRIPE ERROR (update cus plan): ", err)
			return
		}
	}
}

func (wh *stripeWebhook) handleSubCancelled(sub stripe.Subscription) {
	// To prevent from the customer.subscription.updated events
	time.Sleep(15 * time.Second)

	stripeID := sub.Customer.ID

	cus, err := datastore.GetCustomerByStripeID(stripeID)
	if err != nil {
		fmt.Println("STRIPE ERROR (find cus by id): ", err)
		return
	}

	if err := datastore.ChangeCustomerPlan(cus.ID, internal.PlanIdea); err != nil {
		fmt.Println("STRIPE ERROR (update cus plan): ", err)
	}
}

func (wh *stripeWebhook) handlePaymentMethodAttached(pm stripe.PaymentMethod) {
	stripeID := pm.Customer.ID

	cus, err := datastore.GetCustomerByStripeID(stripeID)
	if err != nil {
		fmt.Println("STRIPE ERROR (get cus by stripe id): ", err)
		return
	}

	if cus.IsActive {
		return
	}

	if err := datastore.ActivateCustomer(cus.ID, true); err != nil {
		fmt.Println("STRIPE ERROR (activate cus): ", stripeID, err)
	}
}

func (wh *stripeWebhook) priceToLevel(priceID string) int {
	switch priceID {
	case config.Current.StripePriceIDIdea:
		return internal.PlanIdea
	case config.Current.StripePriceIDLaunch:
		return internal.PleanLaunch
	case config.Current.StripePriceIDTraction:
		return internal.PlanTraction
	case config.Current.StripePriceIDGrowth:
		return internal.PlanGrowth
	default:
		return internal.PlanIdea
	}
}
