package staticbackend

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/model"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/webhook"
)

type stripeWebhook struct {
}

func (wh *stripeWebhook) process(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("STRIPE ERROR (read body)", "error", err)

		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	endpointSecret := config.Current.StripeWebhookSecret

	// Verify webhook signature and extract the event.
	// See https://stripe.com/docs/webhooks/signatures for more information.
	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), endpointSecret)
	if err != nil {
		slog.Error("STRIPE ERROR (verify secret)", "error", err)

		w.WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature.
		return
	}

	switch event.Type {
	case "customer.subscription.updated":
		var sub stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &sub)
		if err != nil {
			slog.Error("STRIPE ERROR (sub update json)", "error", err)

			w.WriteHeader(http.StatusBadRequest)
			return
		}
		go wh.handleSubChanged(sub)
	case "customer.subscription.deleted":
		var sub stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &sub)
		if err != nil {
			slog.Error("STRIPE ERROR (sub del json)", "error", err)

			w.WriteHeader(http.StatusBadRequest)
			return
		}
		go wh.handleSubCancelled(sub)
	case "checkout.session.completed":
		var cs stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &cs); err != nil {
			slog.Error("STRIPE ERROR (checkout session completed JSON)", "error", err)

			w.WriteHeader(http.StatusBadRequest)
			return
		}

		wh.handleCheckoutSessionCompleted(cs)
	default:
		slog.Info("received unhandled Stripe webhook", "type", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

func (wh *stripeWebhook) handleSubChanged(sub stripe.Subscription) {
	if !wh.isSBCustomer(sub.Customer.Metadata) {
		return
	}

	stripeID := sub.Customer.ID

	slog.Info("[Sub Changed]: for StripeID", "stripe_id", stripeID)

	// find the customer
	cus, err := backend.DB.GetTenantByStripeID(stripeID)
	if err != nil {
		slog.Error("STRIPE ERROR (find cus by stripe id)", "stripe_id", stripeID, "error", err)
		return
	}

	slog.Info("[Sub Changed]: found account", "email", cus.Email)

	if sub.Items != nil && len(sub.Items.Data) > 0 {
		slog.Info("[Sub Changed]: there's at least 1 sub")

		priceID := sub.Items.Data[0].Price.ID
		newLevel := wh.priceToLevel(priceID)

		if err := backend.DB.ChangeTenantPlan(cus.ID, newLevel); err != nil {
			slog.Error("STRIPE ERROR (update cus plan)", "error", err)
			return
		}
	}
}

func (wh *stripeWebhook) handleSubCancelled(sub stripe.Subscription) {
	if !wh.isSBCustomer(sub.Customer.Metadata) {
		return
	}

	stripeID := sub.Customer.ID

	cus, err := backend.DB.GetTenantByStripeID(stripeID)
	if err != nil {
		slog.Error("STRIPE ERROR (find cus by id)", "stripe_id", stripeID, "error", err)
		return
	}

	if err := backend.DB.ActivateTenant(cus.ID, false); err != nil {
		slog.Error("STRIPE ERROR (sub canceled)", "tenant_id", cus.ID, "error", err)
	}
}

func (wh *stripeWebhook) handleCheckoutSessionCompleted(cs stripe.CheckoutSession) {
	if !wh.isSBCustomer(cs.Customer.Metadata) {
		slog.Warn("STRIPE: checkout completed, not a sb customer")
		for k, v := range cs.Customer.Metadata {
			slog.Warn("Stripe customer metadata", "key", k, "value", v)
		}
		return
	}

	stripeID := cs.Customer.ID

	cus, err := backend.DB.GetTenantByStripeID(stripeID)
	if err != nil {
		slog.Error("STRIPE ERROR (get cus by stripe id)", "stripe_id", stripeID, "error", err)
		return
	}

	if cus.IsActive {
		return
	}

	if err := backend.DB.ActivateTenant(cus.ID, true); err != nil {
		slog.Error("STRIPE ERROR (activate cus)", "stripe_id", stripeID, "tenant_id", cus.ID, "error", err)
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

func (wh *stripeWebhook) isSBCustomer(m map[string]string) bool {
	v, ok := m["sb"]
	return ok && v == "yes"
}
