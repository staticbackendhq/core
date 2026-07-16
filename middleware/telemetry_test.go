package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/staticbackendhq/core/model"
)

type telemetryCache struct {
	published []model.Command
}

func (c *telemetryCache) Get(key string) (string, error)                                            { return "", nil }
func (c *telemetryCache) Set(key string, value string) error                                        { return nil }
func (c *telemetryCache) Delete(key string) error                                                   { return nil }
func (c *telemetryCache) GetTyped(key string, v any) error                                          { return nil }
func (c *telemetryCache) SetTyped(key string, v any) error                                          { return nil }
func (c *telemetryCache) Inc(key string, by int64) (int64, error)                                   { return 0, nil }
func (c *telemetryCache) Dec(key string, by int64) (int64, error)                                   { return 0, nil }
func (c *telemetryCache) Subscribe(send chan model.Command, token, channel string, close chan bool) {}
func (c *telemetryCache) Publish(msg model.Command) error {
	c.published = append(c.published, msg)
	return nil
}
func (c *telemetryCache) PublishDocument(auth model.Auth, dbname, channel, typ string, v any) {}
func (c *telemetryCache) QueueWork(key, value string) error                                   { return nil }
func (c *telemetryCache) DequeueWork(key string) (string, error)                              { return "", nil }

func TestLongRequestTelemetryDoesNotPublishBelowThreshold(t *testing.T) {
	vol := &telemetryCache{}
	h := LongRequestTelemetry(vol)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := telemetryRequest("/db/tasks", false)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if len(vol.published) != 0 {
		t.Fatalf("expected no telemetry events, got %d", len(vol.published))
	}
}

func TestLongRequestTelemetryPublishesTenantMetadata(t *testing.T) {
	vol := &telemetryCache{}
	h := LongRequestTelemetry(vol)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(SlowRequestThreshold + 10*time.Millisecond)
		http.Error(w, "nope", http.StatusTeapot)
	}))

	req := telemetryRequest("/db/tasks?b=secret&a=hidden", true)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if len(vol.published) != 1 {
		t.Fatalf("expected one telemetry event, got %d", len(vol.published))
	}

	msg := vol.published[0]
	if msg.Channel != model.TelemetryLongRequestChannel {
		t.Fatalf("expected channel %q got %q", model.TelemetryLongRequestChannel, msg.Channel)
	}
	if msg.Type != model.MsgTypeTelemetryLongRequest {
		t.Fatalf("expected type %q got %q", model.MsgTypeTelemetryLongRequest, msg.Type)
	}
	if msg.Base != "testbase" {
		t.Fatalf("expected base testbase got %q", msg.Base)
	}
	if msg.Auth.UserID != "user-1" {
		t.Fatalf("expected auth user user-1 got %q", msg.Auth.UserID)
	}

	var data SlowRequestTelemetry
	if err := json.Unmarshal([]byte(msg.Data), &data); err != nil {
		t.Fatal(err)
	}
	if data.Method != http.MethodGet {
		t.Fatalf("expected method GET got %q", data.Method)
	}
	if data.Path != "/db/tasks" {
		t.Fatalf("expected path /db/tasks got %q", data.Path)
	}
	if data.StatusCode != http.StatusTeapot {
		t.Fatalf("expected status %d got %d", http.StatusTeapot, data.StatusCode)
	}
	if data.DurationMS < SlowRequestThreshold.Milliseconds() {
		t.Fatalf("expected duration over threshold got %dms", data.DurationMS)
	}
	if data.Base != "testbase" || data.AccountID != "account-1" || data.UserID != "user-1" {
		t.Fatalf("unexpected tenant metadata: %+v", data)
	}
	if len(data.QueryKeys) != 2 || data.QueryKeys[0] != "a" || data.QueryKeys[1] != "b" {
		t.Fatalf("expected sorted query keys only, got %+v", data.QueryKeys)
	}
}

func telemetryRequest(target string, withAuth bool) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	ctx := context.WithValue(req.Context(), ContextBase, model.DatabaseConfig{Name: "testbase"})
	if withAuth {
		ctx = context.WithValue(ctx, ContextAuth, model.Auth{
			AccountID: "account-1",
			UserID:    "user-1",
			Token:     "token-1",
		})
	}
	return req.WithContext(ctx)
}
