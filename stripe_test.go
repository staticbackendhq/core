package staticbackend

import (
	"testing"

	"github.com/stripe/stripe-go/v84"
)

func TestHandleCheckoutSessionCompletedWithoutCustomerDoesNotPanic(t *testing.T) {
	wh := stripeWebhook{}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("handleCheckoutSessionCompleted panicked: %v", recovered)
		}
	}()

	wh.handleCheckoutSessionCompleted(stripe.CheckoutSession{})
}
