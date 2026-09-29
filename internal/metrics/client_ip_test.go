package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWhitelistUsesThePeerAddress(t *testing.T) {
	m := NewMetrics()
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	h := m.securityMiddleware(ok, "", []string{"10.0.0.5"})

	spoofed := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	spoofed.RemoteAddr = "203.0.113.9:40000"
	spoofed.Header.Set("X-Forwarded-For", "10.0.0.5")
	spoofed.Header.Set("X-Real-IP", "10.0.0.5")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, spoofed)
	if rec.Code == http.StatusOK {
		t.Error("a forwarded-for header must not get a client past the whitelist")
	}

	allowed := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	allowed.RemoteAddr = "10.0.0.5:40000"
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, allowed)
	if rec.Code != http.StatusOK {
		t.Errorf("a whitelisted peer got %d", rec.Code)
	}

	v6 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	v6.RemoteAddr = "[::1]:40000"
	if got := getClientIP(v6); got != "::1" {
		t.Errorf("IPv6 peer = %q", got)
	}
}
