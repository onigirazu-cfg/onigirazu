package cli

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

type storedServeToken struct {
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	Jobs      []string   `json:"jobs"`
	Groups    []string   `json:"groups"`
	SHA256    string     `json:"sha256"`
	Prefix    string     `json:"prefix"`
	CreatedAt time.Time  `json:"created_at"`
	CreatedBy string     `json:"created_by"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	// Delegated: jobs whose operators list this token passes, because its
	// creator could run them when it was made
	Delegated []string `json:"delegated,omitempty"`
}

type serveTokenStore struct {
	Tokens []storedServeToken `json:"tokens"`
}
type serveTokenState struct {
	sync.Mutex
	tokens      []storedServeToken
	checked     time.Time
	mtime       time.Time
	size        int64
	reloadError string
	usage       map[string]time.Time
	written     time.Time
	// present: tokens.json exists; from then on authentication stays on, even
	// when every token in it expired or was revoked (delete the file to open up)
	present bool
	// writes serializes create and revoke, so file I/O stays outside the mutex
	writes sync.Mutex
}

func readServeTokens(dir string) ([]storedServeToken, error) {
	data, err := os.ReadFile(filepath.Join(dir, "tokens.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var store serveTokenStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, t := range store.Tokens {
		digest, err := hex.DecodeString(t.SHA256)
		if !serveTokenName.MatchString(t.Name) || names[strings.ToLower(t.Name)] || (t.Role != "viewer" && t.Role != "operator") || err != nil || len(digest) != sha256.Size {
			return nil, fmt.Errorf("invalid stored token %q", t.Name)
		}
		names[strings.ToLower(t.Name)] = true
	}
	return store.Tokens, nil
}

func writeServeTokenJSON(path string, value any) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".tokens-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = json.NewEncoder(f).Encode(value); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func lockServeTokens(dir string) (func(), error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "tokens.lock")
	deadline := time.Now().Add(5 * time.Second)
	for {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			_ = f.Close()
			return func() { _ = os.Remove(path) }, nil
		}
		if !os.IsExist(err) {
			return nil, err
		}
		if st, e := os.Stat(path); e == nil && time.Since(st.ModTime()) > 30*time.Second {
			if e = os.Remove(path); e != nil && !os.IsNotExist(e) {
				return nil, e
			}
			continue
		}
		if !time.Now().Before(deadline) {
			return nil, errors.New("timed out waiting for tokens.lock")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func changeServeTokens(cfg *ServeConfig, change func([]storedServeToken) ([]storedServeToken, error)) ([]storedServeToken, error) {
	unlock, err := lockServeTokens(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer unlock()
	tokens, err := readServeTokens(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	tokens, err = change(tokens)
	if err != nil {
		return nil, err
	}
	if tokens == nil {
		tokens = []storedServeToken{}
	}
	err = writeServeTokenJSON(filepath.Join(cfg.DataDir, "tokens.json"), serveTokenStore{Tokens: tokens})
	return tokens, err
}

var errServeTokenDuplicate = errors.New("token name already exists")
var errServeTokenConfig = errors.New("defined in serve.yml")
var errServeTokenMissing = errors.New("unknown token")
var errServeTokenScope = errors.New("token is not owned by caller")

func configServeToken(cfg *ServeConfig, name string) bool {
	return slices.ContainsFunc(cfg.Auth.allTokens(), func(t ServeToken) bool { return strings.EqualFold(t.Name, name) })
}

func addServeToken(cfg *ServeConfig, t storedServeToken) ([]storedServeToken, error) {
	return changeServeTokens(cfg, func(tokens []storedServeToken) ([]storedServeToken, error) {
		if configServeToken(cfg, t.Name) || slices.ContainsFunc(tokens, func(v storedServeToken) bool { return strings.EqualFold(v.Name, t.Name) }) {
			return nil, errServeTokenDuplicate
		}
		return append(tokens, t), nil
	})
}

func revokeServeToken(cfg *ServeConfig, name string, id serveIdentity) ([]storedServeToken, error) {
	return changeServeTokens(cfg, func(tokens []storedServeToken) ([]storedServeToken, error) {
		if configServeToken(cfg, name) {
			return nil, errServeTokenConfig
		}
		i := slices.IndexFunc(tokens, func(t storedServeToken) bool { return strings.EqualFold(t.Name, name) })
		if i < 0 {
			return nil, errServeTokenMissing
		}
		if len(id.Jobs) > 0 && tokens[i].CreatedBy != id.creator() {
			return nil, errServeTokenScope
		}
		return slices.Delete(tokens, i, i+1), nil
	})
}

func makeStoredServeToken(o serveTokenOptions, by string, now func() time.Time) (string, storedServeToken, error) {
	value, t, err := generateServeTokenValue(o, rand.Reader, now)
	if err != nil {
		return "", storedServeToken{}, err
	}
	record := storedServeToken{Name: t.Name, Role: t.Role, Jobs: t.Jobs, Groups: t.Groups, SHA256: strings.TrimPrefix(t.Token, "sha256:"), Prefix: value[4:12], CreatedAt: now().UTC(), CreatedBy: by}
	if !t.expires.IsZero() {
		record.ExpiresAt = &t.expires
	}
	return value, record, nil
}

// creator names who made a token: tokens as token:<name>, so a proxy user
// of the same name does not own what a token created
func (id serveIdentity) creator() string {
	if id.Kind == "token" {
		return "token:" + id.User
	}
	return id.User
}

// delegatedJobs: a new token gets its creator's access to the jobs it is
// for (all jobs when jobs is empty) and to nothing the creator cannot run;
// the jobs with an operators list are recorded, the others need no pass
func (s *server) delegatedJobs(id serveIdentity, jobs []string) ([]string, error) {
	var out []string
	for i := range s.cfg.Jobs {
		j := &s.cfg.Jobs[i]
		wanted := len(jobs) == 0 || slices.Contains(jobs, j.Name)
		if !wanted {
			continue
		}
		if !s.allowed(id, j) {
			if len(jobs) == 0 {
				continue
			}
			return nil, fmt.Errorf("you are not an operator of job %q", j.Name)
		}
		if len(j.Operators) > 0 {
			out = append(out, j.Name)
		}
	}
	return out, nil
}

func validateServeTokenJobs(cfg *ServeConfig, jobs []string) error {
	for _, name := range jobs {
		if !slices.ContainsFunc(cfg.Jobs, func(j ServeJob) bool { return j.Name == name }) {
			return fmt.Errorf("unknown job %q", name)
		}
	}
	return nil
}

// Caller holds tokens.Mutex for reloads, authentication and mutations.
// The error is the reload's; the last good set stays in use.
func (s *server) reloadServeTokens(force bool) error {
	now := s.now()
	if !force && !now.Before(s.tokens.checked) && now.Sub(s.tokens.checked) < 2*time.Second {
		return nil
	}
	s.tokens.checked = now
	st, err := os.Stat(filepath.Join(s.cfg.DataDir, "tokens.json"))
	if os.IsNotExist(err) {
		s.tokens.tokens = nil
		s.tokens.mtime = time.Time{}
		s.tokens.size = 0
		s.tokens.reloadError = ""
		s.tokens.present = false
		return nil
	}
	s.tokens.present = true
	if err == nil && !force && st.ModTime().Equal(s.tokens.mtime) && st.Size() == s.tokens.size {
		return nil
	}
	var tokens []storedServeToken
	if err == nil {
		tokens, err = readServeTokens(s.cfg.DataDir)
	}
	if err == nil {
		for _, t := range tokens {
			if configServeToken(s.cfg, t.Name) {
				err = errServeTokenDuplicate
				break
			}
		}
	}
	if err != nil {
		if s.tokens.reloadError != err.Error() {
			fmt.Fprintf(s.out, "serve: tokens: %v\n", err)
			s.tokens.reloadError = err.Error()
		}
		return err
	}
	s.tokens.tokens = tokens
	s.tokens.mtime = st.ModTime()
	s.tokens.size = st.Size()
	s.tokens.reloadError = ""
	return nil
}

func (s *server) authEnabledLocked() bool {
	if len(s.cfg.Auth.allTokens()) > 0 || s.cfg.Auth.UserHeader != "" {
		return true
	}
	return s.tokens.present
}

func (s *server) authEnabled() bool {
	s.tokens.Lock()
	defer s.tokens.Unlock()
	return s.authEnabledLocked()
}

// tokenUsageLocked copies the usage of tokens that still exist, for a write
// after the mutex is released
func (s *server) tokenUsageLocked() map[string]time.Time {
	known := map[string]bool{}
	for _, t := range s.cfg.Auth.allTokens() {
		known[t.Name] = true
	}
	for _, t := range s.tokens.tokens {
		known[t.Name] = true
	}
	out := map[string]time.Time{}
	for name, at := range s.tokens.usage {
		if known[name] {
			out[name] = at
		}
	}
	s.tokens.written = s.now()
	return out
}

func (s *server) saveTokenUsage(usage map[string]time.Time) {
	if usage == nil {
		return
	}
	if err := writeServeTokenJSON(filepath.Join(s.cfg.DataDir, "token-usage.json"), usage); err != nil {
		fmt.Fprintf(s.out, "serve: token usage: %v\n", err)
	}
}

// useToken records a use; the usage to write when the last write is a minute old
func (s *server) useToken(name string) map[string]time.Time {
	s.tokens.usage[name] = s.now().UTC()
	if s.tokens.written.IsZero() || s.now().Sub(s.tokens.written) >= time.Minute {
		return s.tokenUsageLocked()
	}
	return nil
}

type serveTokenView struct {
	Name       string     `json:"name"`
	Source     string     `json:"source"`
	Role       string     `json:"role"`
	Jobs       []string   `json:"jobs"`
	Groups     []string   `json:"groups"`
	Prefix     string     `json:"prefix,omitempty"`
	CreatedAt  *time.Time `json:"created_at"`
	CreatedBy  string     `json:"created_by"`
	ExpiresAt  *time.Time `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	Expired    bool       `json:"expired"`
	Delegated  []string   `json:"delegated,omitempty"`
}

func (s *server) tokenViewsLocked() []serveTokenView {
	views := make([]serveTokenView, 0)
	for _, t := range s.cfg.Auth.allTokens() {
		if !t.prepared {
			if t.prepare() != nil {
				continue
			}
		}
		v := serveTokenView{Name: t.Name, Source: "config", Role: t.Role, Jobs: t.Jobs, Groups: t.Groups}
		if !t.expires.IsZero() {
			v.ExpiresAt = &t.expires
		}
		views = append(views, v)
	}
	for _, t := range s.tokens.tokens {
		created := t.CreatedAt
		views = append(views, serveTokenView{Name: t.Name, Source: "store", Role: t.Role, Jobs: t.Jobs, Groups: t.Groups, Prefix: t.Prefix, CreatedAt: &created, CreatedBy: t.CreatedBy, ExpiresAt: t.ExpiresAt, Delegated: t.Delegated})
	}
	for i := range views {
		v := &views[i]
		v.Expired = v.ExpiresAt != nil && !s.now().Before(*v.ExpiresAt)
		if used, ok := s.tokens.usage[v.Name]; ok {
			v.LastUsedAt = &used
		}
	}
	slices.SortFunc(views, func(a, b serveTokenView) int { return strings.Compare(a.Name, b.Name) })
	return views
}

func tokenAuditReason(t storedServeToken) string {
	expires := "never"
	if t.ExpiresAt != nil {
		expires = t.ExpiresAt.Format(time.RFC3339)
	}
	return fmt.Sprintf("name=%s role=%s jobs=%s delegated=%s expires=%s", t.Name, t.Role, strings.Join(t.Jobs, ","), strings.Join(t.Delegated, ","), expires)
}

func (s *server) tokensHandler(w http.ResponseWriter, r *http.Request, id serveIdentity) {
	if r.Method == http.MethodGet && r.URL.Path == "/api/tokens" {
		s.tokens.Lock()
		defer s.tokens.Unlock()
		writeJSON(w, 200, map[string]any{"tokens": s.tokenViewsLocked(), "can_create": id.Operator, "auth_enabled": s.authEnabledLocked()})
		return
	}
	create := r.Method == http.MethodPost && r.URL.Path == "/api/tokens"
	revoke := r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/tokens/")
	if !create && !revoke {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	a := s.requestAction(r, id)
	a.Action = "token_create"
	if revoke {
		a.Action = "token_revoke"
		a.Reason = "name=" + strings.TrimPrefix(r.URL.Path, "/api/tokens/")
	}
	defer func() { s.logAction(a) }()
	fail := func(code int, err error) {
		if a.Reason != "" {
			a.Reason += "; "
		}
		a.Reason += err.Error()
		writeJSON(w, code, map[string]string{"error": err.Error()})
	}
	if !id.Operator && r.Method != http.MethodPost {
		fail(403, errors.New("operators only"))
		return
	}
	s.tokens.writes.Lock()
	defer s.tokens.writes.Unlock()
	// an anonymous caller identified before another request turned
	// authentication on is nobody now
	if id.Kind == "none" && s.authEnabled() {
		fail(http.StatusUnauthorized, errors.New("authentication is on now"))
		return
	}
	if create {
		var body struct {
			Name       string          `json:"name"`
			Role       string          `json:"role"`
			Jobs       []string        `json:"jobs"`
			Expires    string          `json:"expires"`
			Groups     json.RawMessage `json:"groups"`
			EnableAuth bool            `json:"enable_auth"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
			fail(400, err)
			return
		}
		a.Reason = tokenAuditReason(storedServeToken{Name: body.Name, Role: body.Role, Jobs: body.Jobs})
		if !id.Operator {
			fail(403, errors.New("operators only"))
			return
		}
		if body.Groups != nil {
			fail(400, errors.New("groups are not accepted"))
			return
		}
		if len(id.Jobs) > 0 && (len(body.Jobs) == 0 || slices.ContainsFunc(body.Jobs, func(j string) bool { return !slices.Contains(id.Jobs, j) })) {
			fail(403, errors.New("jobs must be a nonempty subset of caller jobs"))
			return
		}
		if err := validateServeTokenJobs(s.cfg, body.Jobs); err != nil {
			fail(400, err)
			return
		}
		delegated, err := s.delegatedJobs(id, body.Jobs)
		if err != nil {
			fail(403, err)
			return
		}
		value, record, err := makeStoredServeToken(serveTokenOptions{name: body.Name, role: body.Role, jobs: body.Jobs, expires: body.Expires}, id.creator(), s.now)
		if err != nil {
			fail(400, err)
			return
		}
		record.Delegated = delegated
		// a token does not outlive the one that made it
		if id.Expires != nil && (record.ExpiresAt == nil || record.ExpiresAt.After(*id.Expires)) {
			expires := *id.Expires
			record.ExpiresAt = &expires
		}
		a.Reason = tokenAuditReason(record)
		if !s.authEnabled() && !body.EnableAuth {
			fail(409, errors.New("creating a token turns on authentication; confirm with enable_auth"))
			return
		}
		tokens, err := addServeToken(s.cfg, record)
		s.tokens.Lock()
		if err == nil {
			s.tokens.tokens = tokens
			s.tokens.present = true
		}
		var view serveTokenView
		if err == nil {
			for _, v := range s.tokenViewsLocked() {
				if v.Name == record.Name {
					view = v
					break
				}
			}
		}
		s.tokens.Unlock()
		if err != nil {
			code := 500
			if errors.Is(err, errServeTokenDuplicate) {
				code = 409
			}
			fail(code, err)
			return
		}
		a.OK = true
		writeJSON(w, 201, struct {
			serveTokenView
			Token string `json:"token"`
		}{view, value})
		return
	}
	{
		name := strings.TrimPrefix(r.URL.Path, "/api/tokens/")
		a.Reason = "name=" + name
		s.tokens.Lock()
		for _, t := range s.tokens.tokens {
			if strings.EqualFold(t.Name, name) {
				a.Reason = tokenAuditReason(t)
			}
		}
		s.tokens.Unlock()
		tokens, err := revokeServeToken(s.cfg, name, id)
		s.tokens.Lock()
		if err == nil {
			s.tokens.tokens = tokens
		}
		s.tokens.Unlock()
		if err != nil {
			code := 500
			switch {
			case errors.Is(err, errServeTokenConfig):
				code = 409
			case errors.Is(err, errServeTokenMissing):
				code = 404
			case errors.Is(err, errServeTokenScope):
				code = 403
			}
			fail(code, err)
			return
		}
		a.OK = true
		w.WriteHeader(204)
	}
}
