package winrm

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"

	"github.com/bodgit/ntlmssp"
	ntlmhttp "github.com/bodgit/ntlmssp/http"
	"github.com/masterzen/winrm"
	"github.com/masterzen/winrm/soap"
)

// encryptedNTLM is a WinRM transport with NTLM message encryption (the
// "application/HTTP-SPNEGO-session-encrypted" body pywinrm and Ansible send
// over http). Unlike the library's own Encryption it never falls back to
// unencrypted messages, which a server with AllowUnencrypted=false refuses
// with a bare 401, and it says why authentication failed.
type encryptedNTLM struct {
	user, password string

	mu       sync.Mutex // NTLM sealing numbers messages: one at a time
	url      string
	httpc    *http.Client
	client   *ntlmhttp.Client
	ntlm     *ntlmssp.Client
	endpoint *winrm.Endpoint
}

// Transport prepares the HTTP client for the endpoint
func (e *encryptedNTLM) Transport(endpoint *winrm.Endpoint) error {
	scheme := "http"
	if endpoint.HTTPS {
		scheme = "https"
	}
	e.url = fmt.Sprintf("%s://%s/wsman", scheme, net.JoinHostPort(endpoint.Host, fmt.Sprint(endpoint.Port)))
	e.endpoint = endpoint
	jar, _ := cookiejar.New(nil)
	e.httpc = &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			ResponseHeaderTimeout: endpoint.Timeout,
			MaxIdleConnsPerHost:   1,
		},
	}
	return nil
}

// splitUser reads DOMAIN\user and user@domain
func splitUser(u string) (user, domain string) {
	if d, n, ok := strings.Cut(u, `\`); ok {
		return n, d
	}
	if n, d, ok := strings.Cut(u, "@"); ok {
		return n, d
	}
	return u, ""
}

// authenticate runs the NTLM handshake with an empty message, as pywinrm
// does, so later messages can be sealed
func (e *encryptedNTLM) authenticate() error {
	user, domain := splitUser(e.user)
	ntlmClient, err := ntlmssp.NewClient(ntlmssp.SetUserInfo(user, e.password), ntlmssp.SetDomain(domain),
		ntlmssp.SetVersion(ntlmssp.DefaultVersion()))
	if err != nil {
		return err
	}
	client, err := ntlmhttp.NewClient(e.httpc, ntlmClient, ntlmhttp.Encryption(true))
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, e.url, nil) //nolint:noctx // bounded by the transport's timeouts
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/soap+xml;charset=UTF-8")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("NTLM authentication: %w", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if !ntlmClient.Complete() || resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("NTLM authentication failed: HTTP %d, server offers %q (wrong user or password, or Negotiate off)",
			resp.StatusCode, strings.Join(resp.Header.Values("WWW-Authenticate"), ", "))
	}
	e.client, e.ntlm = client, ntlmClient
	return nil
}

// Post sends one sealed SOAP message and returns the unsealed answer
func (e *encryptedNTLM) Post(_ *winrm.Client, request *soap.SoapMessage) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for attempt := 0; ; attempt++ {
		if e.client == nil {
			if err := e.authenticate(); err != nil {
				return "", err
			}
		}
		req, err := http.NewRequest(http.MethodPost, e.url, bytes.NewBufferString(request.String())) //nolint:noctx // bounded by the transport's timeouts
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/soap+xml;charset=UTF-8")
		resp, err := e.client.Do(req)
		if err != nil {
			e.client = nil // the sealing state is unknown now
			return "", fmt.Errorf("unknown error %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			e.client = nil
			return "", err
		}
		// a new connection lost the NTLM context: authenticate again once
		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			e.client = nil
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("http error %d: %s", resp.StatusCode, body)
		}
		return string(body), nil
	}
}
