package winrm

import (
	"bytes"

	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	ntlmhttp "github.com/bodgit/ntlmssp/http"

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

// the body we send is what Windows (and bodgit's parser) read, with the
// plain message's length in OriginalContent
func TestMimeBody(t *testing.T) {
	payload := []byte("\x10\x00\x00\x00signature-16byt" + "sealed\r\nbytes--")
	body := mimeBody(1234, payload)
	if !bytes.Contains(body, []byte("\tOriginalContent: type=application/soap+xml;charset=UTF-8;Length=1234\r\n")) {
		t.Errorf("body:\n%q", body)
	}
	got, _, err := ntlmhttp.Unwrap(body, encryptedContentType)
	if err != nil || !bytes.Equal(got, payload) {
		t.Errorf("Unwrap = %q, %v", got, err)
	}
}

func TestFaultReason(t *testing.T) {
	body := []byte(`<s:Envelope><s:Body><s:Fault><s:Reason><s:Text xml:lang="en-US">The WS-Management service cannot process the request. </s:Text></s:Reason><s:Detail><f:WSManFault Code="1"><f:Message><f:ProviderFault>Access is denied. </f:ProviderFault></f:Message></f:WSManFault></s:Detail></s:Fault></s:Body></s:Envelope>`)
	if got := faultReason(body); got != "The WS-Management service cannot process the request. | Access is denied. | code 1" {
		t.Errorf("faultReason = %q", got)
	}
}

func TestSoapAction(t *testing.T) {
	if got := soapAction(`<a:Action mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/transfer/Create</a:Action>`); got != "Create" {
		t.Errorf("soapAction = %q", got)
	}
}

// sealed bytes may hold anything: lines that look like headers, CRLFs,
// pieces of the markers; the parts come back exactly
func TestSplitEncrypted(t *testing.T) {
	tricky := []byte("\x10\x00\x00\x00sig\r\n\tX-Fake: header\r\n--Encrypted Bound\n\tContent-Type: nope\r\nend")
	other := []byte("\x10\x00\x00\x00second part \r\n\r\n")
	one := mimeBody(77, tricky)
	two := append(bytes.TrimSuffix(mimeBody(5, tricky), []byte(mimeBoundary+"--\r\n")), mimeBody(9, other)...)
	parts := splitEncrypted(one)
	if len(parts) != 1 || !bytes.Equal(parts[0].payload, tricky) || !bytes.Contains(parts[0].header, []byte("Length=77")) {
		t.Fatalf("one part: %q", parts)
	}
	parts = splitEncrypted(two)
	if len(parts) != 2 || !bytes.Equal(parts[0].payload, tricky) || !bytes.Equal(parts[1].payload, other) ||
		!bytes.Contains(parts[1].header, []byte("Length=9")) {
		t.Fatalf("two parts: %q", parts)
	}
	if len(splitEncrypted([]byte("garbage"))) != 0 {
		t.Error("garbage has no parts")
	}
}
