package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestWithCertificate(t *testing.T) {
	dir := t.TempDir()
	_, userKey, _ := ed25519.GenerateKey(rand.Reader)
	userSigner, _ := ssh.NewSignerFromKey(userKey)
	_, caKey, _ := ed25519.GenerateKey(rand.Reader)
	caSigner, _ := ssh.NewSignerFromKey(caKey)
	cert := &ssh.Certificate{Key: userSigner.PublicKey(), CertType: ssh.UserCert, ValidPrincipals: []string{"deploy"}, ValidBefore: ssh.CertTimeInfinity}
	if err := cert.SignCert(rand.Reader, caSigner); err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(dir, "id_ed25519")
	// no certificate next to the key: the signer is returned as it is
	s, err := withCertificate(userSigner, keyFile, types.Host{})
	if err != nil || s != userSigner {
		t.Fatalf("without a certificate: %v %v", s, err)
	}
	if err := os.WriteFile(keyFile+"-cert.pub", ssh.MarshalAuthorizedKey(cert), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err = withCertificate(userSigner, keyFile, types.Host{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.PublicKey().(*ssh.Certificate); !ok {
		t.Errorf("<key>-cert.pub makes a certificate signer, got %T", s.PublicKey())
	}
	// an explicit certificate that is missing is an error; a named one works
	if _, err := withCertificate(userSigner, keyFile, types.Host{Vars: map[string]interface{}{"ansible_ssh_certificate_file": filepath.Join(dir, "nope")}}); err == nil {
		t.Error("a missing named certificate must fail")
	}
	s, err = withCertificate(userSigner, filepath.Join(dir, "other"), types.Host{Vars: map[string]interface{}{"onigirazu_ssh_certificate_file": keyFile + "-cert.pub"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.PublicKey().(*ssh.Certificate); !ok {
		t.Error("the named certificate is used")
	}
}

func TestAgentAuthMethod(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	if m, _ := agentAuthMethod(nil); m != nil {
		t.Error("no SSH_AUTH_SOCK: no agent auth")
	}
	t.Setenv("SSH_AUTH_SOCK", filepath.Join(t.TempDir(), "none"))
	if m, _ := agentAuthMethod(&nopLogger{}); m != nil {
		t.Error("an agent that cannot be reached is skipped")
	}
}

type nopLogger struct{}

func (nopLogger) Debug(string, ...interface{}) {}
func (nopLogger) Info(string, ...interface{})  {}
func (nopLogger) Warn(string, ...interface{})  {}
func (nopLogger) Error(string, ...interface{}) {}
