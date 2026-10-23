package cli

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const serveSessionCookie = "onigirazu_session"

type serveSessionState struct {
	sync.Mutex
	failures map[string][]time.Time
}

// serveCookieRequest applies to every authenticated API mutation.
func serveCookieRequest(w http.ResponseWriter, r *http.Request, id serveIdentity) bool {
	if id.Via != "cookie" {
		return true
	}
	return sameSiteRequest(w, r, true)
}

// sameSiteRequest refuses a mutation another site's page sent: the sign-in
// itself too, so a page cannot sign the browser in with someone else's token.
// strict (cookie sessions): a request without Origin or Sec-Fetch-Site is
// refused as well, browsers send at least one on a same-origin fetch
func sameSiteRequest(w http.ResponseWriter, r *http.Request, strict bool) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return true
	}
	_, hasSite := r.Header["Sec-Fetch-Site"]
	_, hasOrigin := r.Header["Origin"]
	crossSite := strict && !hasSite && !hasOrigin
	if sites, present := r.Header["Sec-Fetch-Site"]; present {
		crossSite = len(sites) != 1 || sites[0] != "same-origin"
	}
	if origins, present := r.Header["Origin"]; present {
		if len(origins) != 1 {
			crossSite = true
		} else {
			origin, err := url.Parse(origins[0])
			crossSite = crossSite || err != nil || origin.Host == "" || !strings.EqualFold(origin.Host, r.Host)
		}
	}
	if crossSite {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "cross-site request"})
		return false
	}
	return true
}

// clientAddr keys the sign-in limit: behind a listed trusted proxy the
// client it forwarded for (the last X-Forwarded-For entry, the one the proxy
// added), else the peer
func (s *server) clientAddr(r *http.Request, peer string) string {
	if len(s.cfg.Auth.trusted) == 0 || !s.cfg.Auth.fromTrustedProxy(r) {
		return peer
	}
	hops := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	if last := strings.TrimSpace(hops[len(hops)-1]); last != "" {
		return last
	}
	return peer
}

func (s *server) sessionCookie(r *http.Request) *http.Cookie {
	return &http.Cookie{Name: serveSessionCookie, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, // #nosec G124 -- Secure follows the scheme: a plain-HTTP LAN server could not keep the cookie
		Secure: r.TLS != nil || (s.cfg.Auth.fromTrustedProxy(r) && r.Header.Get("X-Forwarded-Proto") == "https")}
}

// signIn serializes failure accounting so concurrent attempts cannot bypass the limit.
func (s *server) signIn(r *http.Request, addr string) (serveIdentity, string, bool, int) {
	var body struct {
		Token    string `json:"token"`
		Remember bool   `json:"remember"`
	}
	decodeErr := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(&body)
	s.sessions.Lock()
	defer s.sessions.Unlock()
	now := s.now()
	if s.sessions.failures == nil {
		s.sessions.failures = map[string][]time.Time{}
	}
	for peer, times := range s.sessions.failures {
		first := 0
		for first < len(times) && !times[first].After(now.Add(-time.Minute)) {
			first++
		}
		if first == len(times) {
			delete(s.sessions.failures, peer)
		} else {
			s.sessions.failures[peer] = times[first:]
		}
	}
	if len(s.sessions.failures[addr]) >= 10 {
		return serveIdentity{}, "", false, http.StatusTooManyRequests
	}
	if decodeErr != nil {
		s.sessions.failures[addr] = append(s.sessions.failures[addr], now)
		return serveIdentity{}, "", false, http.StatusBadRequest
	}
	s.tokens.Lock()
	_ = s.reloadServeTokens(false)
	id, usage := s.tokenIdentityLocked(body.Token, "cookie")
	s.tokens.Unlock()
	s.saveTokenUsage(usage)
	if id.Kind == "" {
		s.sessions.failures[addr] = append(s.sessions.failures[addr], now)
		return id, "", false, http.StatusUnauthorized
	}
	return id, body.Token, body.Remember, http.StatusOK
}

func (s *server) sessionHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		if !sameSiteRequest(w, r, false) {
			return
		}
		a := s.requestAction(r, serveIdentity{})
		id, value, remember, code := s.signIn(r, s.clientAddr(r, a.Addr))
		a.Action, a.User, a.OK = "sign_in", id.User, code == http.StatusOK
		switch code {
		case http.StatusTooManyRequests:
			a.Reason = "too many attempts, try again in a minute"
		case http.StatusBadRequest:
			a.Reason = "invalid JSON"
		case http.StatusUnauthorized:
			a.Reason = "unknown or expired token"
		}
		s.logAction(a)
		if code != http.StatusOK {
			writeJSON(w, code, map[string]string{"error": a.Reason})
			return
		}
		cookie := s.sessionCookie(r) // #nosec G124 -- see sessionCookie
		cookie.Value = value
		if remember {
			cookie.MaxAge = 30 * 24 * 60 * 60
			if id.Expires != nil {
				cookie.MaxAge = min(cookie.MaxAge, int(id.Expires.Sub(s.now())/time.Second))
				if cookie.MaxAge <= 0 {
					cookie.MaxAge = -1
				}
			}
		}
		http.SetCookie(w, cookie)
		s.meHandler(w, r, id)
	case http.MethodDelete:
		id, err := s.identify(r)
		// signing out is harmless: only a cross-site request is refused
		if !sameSiteRequest(w, r, false) {
			return
		}
		cookie := s.sessionCookie(r) // #nosec G124 -- see sessionCookie
		cookie.MaxAge = -1           // net/http writes Max-Age=0 to delete the cookie.
		http.SetCookie(w, cookie)
		if err == nil {
			a := s.requestAction(r, id)
			a.Action, a.OK = "sign_out", true
			s.logAction(a)
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "POST, DELETE")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
