package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/model"
)

const SlowRequestThreshold = 150 * time.Millisecond

type SlowRequestTelemetry struct {
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	QueryKeys  []string  `json:"queryKeys,omitempty"`
	StatusCode int       `json:"statusCode"`
	DurationMS int64     `json:"durationMs"`
	Base       string    `json:"base"`
	AccountID  string    `json:"accountId,omitempty"`
	UserID     string    `json:"userId,omitempty"`
	Started    time.Time `json:"started"`
	Completed  time.Time `json:"completed"`
}

type telemetryResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *telemetryResponseWriter) WriteHeader(code int) {
	if w.statusCode != 0 {
		return
	}
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *telemetryResponseWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// LongRequestTelemetry publishes a tenant event for requests exceeding
// SlowRequestThreshold. Publishing is best effort and does not affect responses.
func LongRequestTelemetry(volatile cache.Volatilizer) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			tw := &telemetryResponseWriter{ResponseWriter: w}
			ww := http.ResponseWriter(tw)
			if _, ok := w.(http.Flusher); ok {
				ww = &telemetryFlushResponseWriter{telemetryResponseWriter: tw}
			}

			defer func() {
				completed := time.Now()
				duration := completed.Sub(started)
				if duration <= SlowRequestThreshold {
					return
				}

				conf, ok := r.Context().Value(ContextBase).(model.DatabaseConfig)
				if !ok {
					return
				}

				auth, _ := r.Context().Value(ContextAuth).(model.Auth)
				statusCode := tw.statusCode
				if statusCode == 0 {
					statusCode = http.StatusOK
				}

				data := SlowRequestTelemetry{
					Method:     r.Method,
					Path:       r.URL.Path,
					QueryKeys:  queryKeys(r),
					StatusCode: statusCode,
					DurationMS: duration.Milliseconds(),
					Base:       conf.Name,
					AccountID:  auth.AccountID,
					UserID:     auth.UserID,
					Started:    started.UTC(),
					Completed:  completed.UTC(),
				}

				b, err := json.Marshal(data)
				if err != nil {
					slog.Error("error marshaling slow request telemetry", "error", err)
					return
				}

				msg := model.Command{
					SID:     "telemetry",
					Type:    model.MsgTypeTelemetryLongRequest,
					Data:    string(b),
					Channel: model.TelemetryLongRequestChannel,
					Token:   auth.Token,
					Auth:    auth,
					Base:    conf.Name,
				}
				if err := volatile.Publish(msg); err != nil {
					slog.Error("error publishing slow request telemetry", "error", err)
				}
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

type telemetryFlushResponseWriter struct {
	*telemetryResponseWriter
}

func (w *telemetryFlushResponseWriter) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

func queryKeys(r *http.Request) []string {
	values := r.URL.Query()
	if len(values) == 0 {
		return nil
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
