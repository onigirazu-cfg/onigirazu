package winrm

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/masterzen/winrm"
	"github.com/masterzen/winrm/soap"
)

func TestSplitUser(t *testing.T) {
	for in, want := range map[string][2]string{
		`OFFICE\bob`: {"bob", "OFFICE"}, "bob@office.red": {"bob", "office.red"}, "e2e": {"e2e", ""},
	} {
		if u, d := splitUser(in); u != want[0] || d != want[1] {
			t.Errorf("splitUser(%q) = %q, %q", in, u, d)
		}
	}
}

// a server that never accepts the credentials: the error names the HTTP
// status and what the server offers, and nothing goes out unencrypted
func TestEncryptedNTLMRefused(t *testing.T) {
	var plain int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > 0 && !strings.Contains(r.Header.Get("Content-Type"), "multipart/encrypted") {
			plain++
		}
		w.Header().Set("WWW-Authenticate", "Negotiate")
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()
	host, port, _ := net.SplitHostPort(strings.TrimPrefix(ts.URL, "http://"))
	p, _ := strconv.Atoi(port)
	e := &encryptedNTLM{user: "e2e", password: "x"}
	if err := e.Transport(&winrm.Endpoint{Host: host, Port: p, Timeout: 5 * time.Second}); err != nil {
		t.Fatal(err)
	}
	_, err := e.Post(nil, soap.NewMessage())
	if err == nil || !strings.Contains(err.Error(), "NTLM authentication") {
		t.Errorf("error = %v", err)
	}
	if plain != 0 {
		t.Errorf("%d unencrypted messages sent", plain)
	}
}
