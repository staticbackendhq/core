package staticbackend

import (
	"testing"

	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/logger"
	"github.com/stripe/stripe-go/v84"
)

func TestHandleCheckoutSessionCompletedWithoutCustomerDoesNotPanic(t *testing.T) {
	wh := stripeWebhook{log: logger.Get(config.AppConfig{})}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("handleCheckoutSessionCompleted panicked: %v", recovered)
		}
	}()

	wh.handleCheckoutSessionCompleted(stripe.CheckoutSession{})
}
