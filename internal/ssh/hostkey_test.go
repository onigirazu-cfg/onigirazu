package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// testingTB is an interface that covers both *testing.T and *testing.B
type testingTB interface {
	Helper()
	Fatalf(format string, args ...interface{})
}

// generateTestKey generates a test SSH key pair and returns the public key
func generateTestKey(t testingTB) ssh.PublicKey {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	return signer.PublicKey()
}

// generateTestPrivateKey generates a test SSH private key and returns it in OpenSSH format
func generateTestPrivateKey(t testingTB) []byte {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Marshal the private key to OpenSSH format
	pemBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		t.Fatalf("failed to marshal private key: %v", err)
	}

	// Encode the PEM block to bytes
	return pem.EncodeToMemory(pemBlock)
}

// TestNewHostKeyManager tests creating a new host key manager

var testAddr = &net.TCPAddr{IP: net.ParseIP("192.0.2.10"), Port: 22}

func writeKnownHosts(t *testing.T, lines ...string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "known_hosts")
	require.NoError(t, os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600))
	return p
}

func TestHostKeyKnownHostAndChangedKey(t *testing.T) {
	key, other := generateTestKey(t), generateTestKey(t)
	file := writeKnownHosts(t, "# comment", knownhosts.Line([]string{"web1"}, key))
	m := NewHostKeyManager(file, false)
	assert.NoError(t, m.VerifyHostKey("web1:22", testAddr, key))
	err := m.VerifyHostKey("web1:22", testAddr, other)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "changed")
}

func TestHostKeyNewHostIsAddedInOpenSSHFormat(t *testing.T) {
	key := generateTestKey(t)
	file := filepath.Join(t.TempDir(), "sub", "known_hosts")
	m := NewHostKeyManager(file, false)
	require.NoError(t, m.VerifyHostKey("127.0.0.1:2222", testAddr, key))
	require.NoError(t, m.VerifyHostKey("127.0.0.1:2222", testAddr, key), "the same run accepts it again")
	assert.Error(t, m.VerifyHostKey("127.0.0.1:2222", testAddr, generateTestKey(t)))

	data, err := os.ReadFile(file) // #nosec G304 -- test file
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.Len(t, lines, 1, "one line per new host")
	assert.Equal(t, knownhosts.Line([]string{"[127.0.0.1]:2222"}, key), lines[0])
	assert.Len(t, strings.Fields(lines[0]), 3, "host, key type, key: the type is not repeated")

	// a new run reads what it wrote
	assert.NoError(t, NewHostKeyManager(file, true).VerifyHostKey("127.0.0.1:2222", testAddr, key))
	info, _ := os.Stat(file)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestHostKeyStrictModeRejectsUnknown(t *testing.T) {
	file := writeKnownHosts(t)
	err := NewHostKeyManager(file, true).VerifyHostKey("new:22", testAddr, generateTestKey(t))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown host")
}

func TestHostKeyHashedAndPortEntries(t *testing.T) {
	key := generateTestKey(t)
	hashed := knownhosts.Line([]string{knownhosts.HashHostname("db1")}, key)
	file := writeKnownHosts(t, hashed, knownhosts.Line([]string{"[10.0.0.5]:2200"}, key))
	m := NewHostKeyManager(file, true)
	assert.NoError(t, m.VerifyHostKey("db1:22", testAddr, key), "hashed name")
	assert.NoError(t, m.VerifyHostKey("10.0.0.5:2200", testAddr, key), "[host]:port")
	assert.Error(t, m.VerifyHostKey("10.0.0.5:22", testAddr, key), "another port is another host")
}

func TestHostKeyMalformedLinesDoNotBreakLoading(t *testing.T) {
	key := generateTestKey(t)
	// the lines older versions wrote: the key type twice
	old := "web1 " + key.Type() + " " + strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
	file := writeKnownHosts(t, old, knownhosts.Line([]string{"web2"}, key))
	m := NewHostKeyManager(file, true)
	assert.NoError(t, m.VerifyHostKey("web2:22", testAddr, key))
}

func TestHostKeyHomeInPathAndInsecure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	m := NewHostKeyManager("~/.ssh/kh", false)
	require.NoError(t, m.VerifyHostKey("h:22", testAddr, generateTestKey(t)))
	_, err := os.Stat(filepath.Join(home, ".ssh", "kh"))
	assert.NoError(t, err, "~ is the home directory")

	insecure := NewHostKeyManagerWithInsecure(filepath.Join(home, "none"), true, true)
	assert.NoError(t, insecure.VerifyHostKey("any:22", testAddr, generateTestKey(t)))
}

func TestHostKeyCallbackHonoursInsecureHost(t *testing.T) {
	m := NewHostKeyManager(writeKnownHosts(t), true)
	cb := hostKeyCallback(types.Host{Name: "h", InsecureIgnoreHostKey: true}, m)
	assert.NoError(t, cb("h:22", testAddr, generateTestKey(t)))
	assert.Error(t, hostKeyCallback(types.Host{Name: "h"}, m)("h:22", testAddr, generateTestKey(t)))
}

func TestGetFingerprint(t *testing.T) {
	key := generateTestKey(t)
	m := NewHostKeyManager(writeKnownHosts(t), false)
	assert.Equal(t, ssh.FingerprintSHA256(key), m.GetFingerprint(key))
}
