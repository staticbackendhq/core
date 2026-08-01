package staticbackend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/staticbackendhq/core/config"
)

func TestLoginHidesCustomerCreationWhenDisabled(t *testing.T) {
	if err := loadTemplates(); err != nil {
		t.Fatal(err)
	}

	original := config.Current.NoCustomerCreation
	config.Current.NoCustomerCreation = true
	t.Cleanup(func() { config.Current.NoCustomerCreation = original })

	resp := httptest.NewRecorder()
	(&ui{}).login(resp, httptest.NewRequest(http.MethodGet, "/", nil))

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	body := resp.Body.String()
	if strings.Contains(body, "action=\"/account/init\"") {
		t.Fatal("expected customer creation form to be hidden")
	}
	if !strings.Contains(body, "New app creation is unavailable") {
		t.Fatal("expected customer creation unavailable message")
	}
	if !strings.Contains(body, "action=\"/ui/login\"") {
		t.Fatal("expected sign-in form to remain available")
	}
}
