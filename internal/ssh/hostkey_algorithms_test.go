package ssh

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// a server with ecdsa and ed25519 host keys, known_hosts knowing only ed25519:
// OpenSSH asks for the known type; so must we, or the server sends ecdsa
func TestHostKeyAlgorithmsFollowKnownHosts(t *testing.T) {
	_, edPriv, _ := ed25519.GenerateKey(rand.Reader)
	ecPriv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	edSigner, _ := ssh.NewSignerFromKey(edPriv)
	ecSigner, _ := ssh.NewSignerFromKey(ecPriv)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	server := &ssh.ServerConfig{NoClientAuth: true}
	server.AddHostKey(ecSigner)
	server.AddHostKey(edSigner)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				_, chans, reqs, err := ssh.NewServerConn(c, server)
				if err != nil {
					return
				}
				go ssh.DiscardRequests(reqs)
				for ch := range chans {
					_ = ch.Reject(ssh.Prohibited, "test")
				}
			}()
		}
	}()

	address := ln.Addr().String()
	file := filepath.Join(t.TempDir(), "known_hosts")
	line := knownhosts.Line([]string{knownhosts.Normalize(address)}, edSigner.PublicKey())
	if err := os.WriteFile(file, []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := NewHostKeyManager(file, true)
	if got := m.HostKeyAlgorithms(address); !slices.Equal(got, []string{ssh.KeyAlgoED25519}) {
		t.Fatalf("algorithms: %v", got)
	}
	if got := m.HostKeyAlgorithms("192.0.2.9:22"); got != nil {
		t.Fatalf("an unknown host keeps the library order: %v", got)
	}

	dial := func(algos []string) error {
		c, err := ssh.Dial("tcp", address, &ssh.ClientConfig{User: "u", HostKeyCallback: m.VerifyHostKey, HostKeyAlgorithms: algos})
		if err == nil {
			_ = c.Close()
		}
		return err
	}
	if err := dial(nil); err == nil {
		t.Fatal("without the known types the server presents ecdsa and the check must fail (the bug this guards)")
	}
	if err := dial(m.HostKeyAlgorithms(address)); err != nil {
		t.Fatalf("with the known types: %v", err)
	}
}

func TestHostKeyAlgorithmsRSA(t *testing.T) {
	got := hostKeyAlgorithms([]string{ssh.KeyAlgoRSA, ssh.KeyAlgoED25519, ssh.KeyAlgoRSA})
	want := []string{ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSA, ssh.KeyAlgoED25519}
	if !slices.Equal(got, want) {
		t.Fatalf("%v", got)
	}
}

// a host trusted through @cert-authority keeps the library's order (host
// certificates first): the CA's key type says nothing about the host's
func TestHostKeyAlgorithmsCertAuthority(t *testing.T) {
	_, caPriv, _ := ed25519.GenerateKey(rand.Reader)
	ca, _ := ssh.NewSignerFromKey(caPriv)
	file := filepath.Join(t.TempDir(), "known_hosts")
	line := "@cert-authority *.example.com " + string(ssh.MarshalAuthorizedKey(ca.PublicKey()))
	if err := os.WriteFile(file, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	m := NewHostKeyManager(file, true)
	if got := m.HostKeyAlgorithms("web1.example.com:22"); got != nil {
		t.Fatalf("CA-trusted host: %v", got)
	}
}
