package winrm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
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
// with a bare 401, and it says why authentication or a request failed.
type encryptedNTLM struct {
	user, password string
	url            string
	endpoint       *winrm.Endpoint
}

// Transport records the endpoint
func (e *encryptedNTLM) Transport(endpoint *winrm.Endpoint) error {
	scheme := "http"
	if endpoint.HTTPS {
		scheme = "https"
	}
	e.url = fmt.Sprintf("%s://%s/wsman", scheme, net.JoinHostPort(endpoint.Host, fmt.Sprint(endpoint.Port)))
	e.endpoint = endpoint
	return nil
}

// connection is one TCP connection with its NTLM session. The library sends
// stdin while it polls for output, so every message gets its own connection
// and handshake (as the library's own Encryption does): a shared sealed
// connection would queue a Send behind a Receive that waits for its input.
type connection struct {
	httpc *http.Client
	ntlm  *ntlmssp.Client
}

func (e *encryptedNTLM) newHTTPClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			ResponseHeaderTimeout: e.endpoint.Timeout + 30*time.Second,
			MaxConnsPerHost:       1,
		},
	}
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
func (e *encryptedNTLM) authenticate() (*connection, error) {
	user, domain := splitUser(e.user)
	ntlmClient, err := ntlmssp.NewClient(ntlmssp.SetUserInfo(user, e.password), ntlmssp.SetDomain(domain),
		ntlmssp.SetVersion(ntlmssp.DefaultVersion()))
	if err != nil {
		return nil, err
	}
	httpc := e.newHTTPClient()
	client, err := ntlmhttp.NewClient(httpc, ntlmClient, ntlmhttp.Encryption(true))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, e.url, nil) //nolint:noctx // bounded by the transport's timeouts
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", soapContentType)
	resp, err := client.Do(req)
	if err != nil {
		httpc.CloseIdleConnections()
		return nil, fmt.Errorf("NTLM authentication: %w", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if !ntlmClient.Complete() || resp.StatusCode == http.StatusUnauthorized {
		httpc.CloseIdleConnections()
		return nil, fmt.Errorf("NTLM authentication failed: HTTP %d, server offers %q (wrong user or password, or Negotiate off)",
			resp.StatusCode, strings.Join(resp.Header.Values("WWW-Authenticate"), ", "))
	}
	return &connection{httpc: httpc, ntlm: ntlmClient}, nil
}

// seal turns a SOAP message into the encrypted MIME body pywinrm sends:
// OriginalContent carries the length of the plain message (bodgit's own
// Wrap writes the sealed length, which Windows answers with 400)
func (c *connection) seal(message []byte) ([]byte, string, error) {
	session := c.ntlm.SecuritySession()
	if session == nil {
		return nil, "", errors.New("no NTLM security session")
	}
	sealed, signature, err := session.Wrap(message)
	if err != nil {
		return nil, "", err
	}
	length := make([]byte, 4)
	binary.LittleEndian.PutUint32(length, uint32(len(signature))) // #nosec G115 -- an NTLM signature is 16 bytes
	return mimeBody(len(message), bytes.Join([][]byte{length, signature, sealed}, nil)), encryptedContentType, nil
}

// mimeBody wraps the encrypted payload of a message of plainLength bytes
func mimeBody(plainLength int, payload []byte) []byte {
	var b bytes.Buffer
	b.WriteString(mimeBoundary + "\r\n")
	b.WriteString("\tContent-Type: application/HTTP-SPNEGO-session-encrypted\r\n")
	fmt.Fprintf(&b, "\tOriginalContent: type=%s;Length=%d\r\n", soapContentType, plainLength)
	b.WriteString(mimeBoundary + "\r\n")
	b.WriteString("\tContent-Type: application/octet-stream\r\n")
	b.Write(payload)
	b.WriteString(mimeBoundary + "--\r\n")
	return b.Bytes()
}

const (
	mimeBoundary         = "--Encrypted Boundary"
	encryptedContentType = `multipart/encrypted;protocol="application/HTTP-SPNEGO-session-encrypted";boundary="Encrypted Boundary"`
)

// unseal returns the SOAP message of an encrypted answer
func (c *connection) unseal(body []byte, contentType string) ([]byte, error) {
	data, _, err := ntlmhttp.Unwrap(body, contentType)
	if err != nil {
		return nil, err
	}
	if len(data) < 4 {
		return nil, errors.New("encrypted answer too short")
	}
	n := int(binary.LittleEndian.Uint32(data[:4]))
	if n > len(data)-4 {
		return nil, errors.New("encrypted answer: bad signature length")
	}
	return c.ntlm.SecuritySession().Unwrap(data[4+n:], data[4:4+n])
}

const soapContentType = "application/soap+xml;charset=UTF-8"

// Post sends one sealed SOAP message on its own authenticated connection
// and returns the unsealed answer
func (e *encryptedNTLM) Post(_ *winrm.Client, request *soap.SoapMessage) (string, error) {
	conn, err := e.authenticate()
	if err != nil {
		return "", err
	}
	defer conn.httpc.CloseIdleConnections()
	body, contentType, err := conn.seal([]byte(request.String()))
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, e.url, bytes.NewReader(body)) //nolint:noctx // bounded by the transport's timeouts
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", contentType)
	// the handshake's keep-alive connection: NTLM authenticates connections
	resp, err := conn.httpc.Do(req)
	if err != nil {
		return "", fmt.Errorf("unknown error %w", err)
	}
	answer, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return "", err
	}
	if ct := resp.Header.Get("Content-Type"); strings.HasPrefix(ct, "multipart/encrypted") {
		// the reply's Content-Type may differ in spelling from the one we
		// send; Unwrap compares it exactly
		if answer, err = conn.unseal(answer, contentType); err != nil {
			return "", fmt.Errorf("decrypting the answer: %w", err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		// the fault's reason first; the body stays for the library, which
		// looks for "OperationTimeout" in it
		return "", fmt.Errorf("http error %d: %s (%s, %d bytes sent): %s", resp.StatusCode, faultReason(answer),
			soapAction(request.String()), len(body), answer)
	}
	return string(answer), nil
}

var faultText = regexp.MustCompile(`(?s)<(?:s:Text|f:Message)[^>]*>(.*?)</(?:s:Text|f:Message)>`)

// faultReason is the human text of a WS-Management fault
func faultReason(body []byte) string {
	var parts []string
	for _, m := range faultText.FindAllSubmatch(body, -1) {
		t := strings.TrimSpace(regexp.MustCompile(`<[^>]+>`).ReplaceAllString(string(m[1]), " "))
		if t != "" && !strings.Contains(strings.Join(parts, " "), t) {
			parts = append(parts, t)
		}
	}
	if m := faultCode.FindSubmatch(body); m != nil {
		parts = append(parts, "code "+string(m[1]))
	}
	if len(parts) == 0 {
		return "WS-Management fault"
	}
	return strings.Join(parts, " | ")
}

var (
	faultCode = regexp.MustCompile(`WSManFault[^>]*Code="(\d+)"`)
	soapActRe = regexp.MustCompile(`<a:Action[^>]*>[^<]*/([A-Za-z]+)</a:Action>`)
)

// soapAction is the last part of a request's WS-Addressing action
// (Create, Command, Send, Receive, Signal, Delete)
func soapAction(message string) string {
	if m := soapActRe.FindStringSubmatch(message); m != nil {
		return m[1]
	}
	return "request"
}
