package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Roma7-7-7/english-learning-bot/internal/api"
	"github.com/Roma7-7-7/english-learning-bot/internal/config"
)

// TestHealthReportsBuildInfo pins the /health payload: the web UI navbar reads these exact field
// names to show which build is running, so renaming one here silently breaks the tooltip.
func TestHealthReportsBuildInfo(t *testing.T) {
	conf := &config.Bot{}
	conf.HTTP.RateLimit = 100
	conf.HTTP.ProcessTimeout = 10 * time.Second
	conf.HTTP.CORS.AllowOrigins = []string{"http://localhost:3000"}
	conf.HTTP.Cookie.Domain = "localhost"
	conf.HTTP.JWT.Audience = []string{"http://localhost:3000"}
	conf.HTTP.JWT.Secret = "test-secret"
	conf.BuildInfo.Version = "a1b2c3d"
	conf.BuildInfo.BuildTime = "20260809-143513"

	handler := api.NewRouter(context.Background(), conf, api.Dependencies{
		Logger: testLogger(),
	})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal body %q: %v", rec.Body.String(), err)
	}

	want := map[string]string{
		"status":     "ok",
		"version":    "a1b2c3d",
		"build_time": "20260809-143513",
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%q = %q, want %q", key, got[key], value)
		}
	}
}
