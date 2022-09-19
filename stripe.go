package staticbackend

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/webhook"
)

type stripeWebhook struct {
	log *logger.Logger
}

func (wh *stripeWebhook) process(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		wh.log.Error().Err(err).Msg("STRIPE ERROR (read body)")

		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	endpointSecret := config.Current.StripeWebhookSecret

	// Verify webhook signature and extract the event.
	// See https://stripe.com/docs/webhooks/signatures for more information.
	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), endpointSecret)
	if err != nil {
		wh.log.Error().Err(err).Msg("STRIPE ERROR (verify secret)")

		w.WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature.
		return
	}

	if event.Type == "customer.subscription.updated" {
		var sub stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &sub)
		if err != nil {
			wh.log.Error().Err(err).Msg("STRIPE ERROR (sub update json))")

			w.WriteHeader(http.StatusBadRequest)
			return
		}
		go wh.handleSubChanged(sub)
	} else if event.Type == "customer.subscription.deleted" {
		var sub stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &sub)
		if err != nil {
			wh.log.Error().Err(err).Msg("STRIPE ERROR (sub del json))")

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

	wh.log.Info().Msgf("[Sub Changed]: for StripeID: %d", stripeID)

	// find the customer
	cus, err := backend.DB.GetTenantByStripeID(stripeID)
	if err != nil {
		wh.log.Error().Err(err).Msg("STRIPE ERROR (find cus by stripe id)")
		return
	}

	wh.log.Info().Msgf("[Sub Changed]: found account: %s", cus.Email)

	if sub.Items.TotalCount > 0 {
		wh.log.Info().Msg("[Sub Changed]: there's at least 1 sub")

		priceID := sub.Items.Data[0].Price.ID
		newLevel := wh.priceToLevel(priceID)

		if err := backend.DB.ChangeTenantPlan(cus.ID, newLevel); err != nil {
			wh.log.Error().Err(err).Msg("STRIPE ERROR (update cus plan)")
			return
		}
	}
}

func (wh *stripeWebhook) handleSubCancelled(sub stripe.Subscription) {
	// To prevent from the customer.subscription.updated events
	time.Sleep(15 * time.Second)

	stripeID := sub.Customer.ID

	cus, err := backend.DB.GetTenantByStripeID(stripeID)
	if err != nil {
		wh.log.Error().Err(err).Msg("STRIPE ERROR (find cus by id)")
		return
	}

	if err := backend.DB.ChangeTenantPlan(cus.ID, model.PlanIdea); err != nil {
		wh.log.Error().Err(err).Msg("STRIPE ERROR (update cus plan)")
	}
}

func (wh *stripeWebhook) handlePaymentMethodAttached(pm stripe.PaymentMethod) {
	stripeID := pm.Customer.ID

	cus, err := backend.DB.GetTenantByStripeID(stripeID)
	if err != nil {
		wh.log.Error().Err(err).Msg("STRIPE ERROR (get cus by stripe id)")
		return
	}

	if cus.IsActive {
		return
	}

	if err := backend.DB.ActivateTenant(cus.ID, true); err != nil {
		wh.log.Error().Err(err).Msgf("STRIPE ERROR (activate cus): %d", stripeID)
	}
}

func (wh *stripeWebhook) priceToLevel(priceID string) int {
	switch priceID {
	case config.Current.StripePriceIDIdea:
		return model.PlanIdea
	case config.Current.StripePriceIDLaunch:
		return model.PleanLaunch
	case config.Current.StripePriceIDTraction:
		return model.PlanTraction
	case config.Current.StripePriceIDGrowth:
		return model.PlanGrowth
	default:
		return model.PlanIdea
	}
}
