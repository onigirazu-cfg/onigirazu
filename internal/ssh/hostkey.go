package ssh

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// keysEqual compares two SSH public keys
func keysEqual(a, b ssh.PublicKey) bool {
	return bytes.Equal(a.Marshal(), b.Marshal())
}

// HostKeyManager checks host keys against an OpenSSH known_hosts file:
// a known host must present its key, a new host's key is added (unless
// strict), and a changed key fails the connection
type HostKeyManager struct {
	knownHostsFile string
	callback       ssh.HostKeyCallback // from the file as it was loaded
	authorities    map[string]bool     // @cert-authority keys of the file, by wire form
	added          map[string]ssh.PublicKey
	mutex          sync.RWMutex
	strictMode     bool
	insecure       bool
}

func NewHostKeyManager(knownHostsFile string, strictMode bool) *HostKeyManager {
	return NewHostKeyManagerWithInsecure(knownHostsFile, strictMode, false)
}

func NewHostKeyManagerWithInsecure(knownHostsFile string, strictMode bool, insecure bool) *HostKeyManager {
	if knownHostsFile == "" {
		home, _ := os.UserHomeDir()
		knownHostsFile = filepath.Join(home, ".ssh", "known_hosts")
	} else if strings.HasPrefix(knownHostsFile, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			knownHostsFile = filepath.Join(home, knownHostsFile[2:])
		}
	}

	hkm := &HostKeyManager{
		knownHostsFile: knownHostsFile,
		added:          make(map[string]ssh.PublicKey),
		strictMode:     strictMode,
		insecure:       insecure,
	}

	if !insecure {
		_ = hkm.loadKnownHosts()
	}
	return hkm
}

// loadKnownHosts reads the file with OpenSSH's rules: hashed names,
// [host]:port, several names per line, @revoked; a missing file is empty
func (hkm *HostKeyManager) loadKnownHosts() error {
	hkm.mutex.Lock()
	defer hkm.mutex.Unlock()
	if _, err := os.Stat(hkm.knownHostsFile); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	data, err := os.ReadFile(hkm.knownHostsFile)
	if err != nil {
		return err
	}
	// knownhosts rejects a whole file for one bad line; OpenSSH skips it.
	// Older onigirazu versions wrote such lines (the key type twice).
	tmp, err := os.CreateTemp("", "onigirazu-known-hosts-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	authorities := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		if validKnownHostsLine(line) {
			_, _ = tmp.WriteString(line + "\n")
			if f := strings.Fields(line); f[0] == "@cert-authority" {
				if key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(f[2] + " " + f[3])); err == nil {
					authorities[string(key.Marshal())] = true
				}
			}
		}
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	cb, err := knownhosts.New(tmp.Name())
	if err != nil {
		return err
	}
	hkm.callback = cb
	hkm.authorities = authorities
	return nil
}

// validKnownHostsLine tells whether a known_hosts line holds a key:
// [@marker] hosts keytype base64 [comment]
func validKnownHostsLine(line string) bool {
	fields := strings.Fields(line)
	if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
		return false
	}
	if strings.HasPrefix(fields[0], "@") {
		fields = fields[1:]
	}
	if len(fields) < 3 {
		return false
	}
	_, _, _, _, err := ssh.ParseAuthorizedKey([]byte(fields[1] + " " + fields[2]))
	return err == nil
}

// VerifyHostKey is the ssh.HostKeyCallback of the manager
func (hkm *HostKeyManager) VerifyHostKey(hostname string, remote net.Addr, key ssh.PublicKey) error {
	if hkm.insecure {
		return nil
	}
	name := knownhosts.Normalize(hostname)

	hkm.mutex.RLock()
	cb := hkm.callback
	addedKey, wasAdded := hkm.added[name]
	hkm.mutex.RUnlock()

	if wasAdded {
		if keysEqual(key, addedKey) {
			return nil
		}
		return fmt.Errorf("host key verification failed for %s: the key changed", hostname)
	}
	if cb != nil {
		err := cb(hostname, remote, key)
		if err == nil {
			return nil
		}
		var keyErr *knownhosts.KeyError
		if !errors.As(err, &keyErr) || len(keyErr.Want) > 0 {
			// a known host with another key, or a revoked key
			return fmt.Errorf("host key verification failed for %s: %w (the host key changed; if this is expected, remove the old line from %s)", hostname, err, hkm.knownHostsFile)
		}
	}
	if hkm.strictMode {
		return fmt.Errorf("host key verification failed for %s: unknown host (ssh_strict_host_key is on; add its key to %s)", hostname, hkm.knownHostsFile)
	}
	return hkm.addHostKey(name, key)
}

// addHostKey appends an OpenSSH known_hosts line for a new host
func (hkm *HostKeyManager) addHostKey(name string, key ssh.PublicKey) error {
	hkm.mutex.Lock()
	defer hkm.mutex.Unlock()
	if _, ok := hkm.added[name]; ok {
		return nil
	}
	hkm.added[name] = key

	if err := os.MkdirAll(filepath.Dir(hkm.knownHostsFile), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(hkm.knownHostsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.WriteString(knownhosts.Line([]string{name}, key) + "\n"); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func (hkm *HostKeyManager) GetFingerprint(key ssh.PublicKey) string {
	// Use SHA256 instead of MD5 for better security
	hash := sha256.Sum256(key.Marshal())
	// Return base64-encoded SHA256 fingerprint (modern OpenSSH format)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(hash[:])
}

func (hkm *HostKeyManager) IsStrictMode() bool {
	return hkm.strictMode
}

func (hkm *HostKeyManager) SetStrictMode(strict bool) {
	hkm.mutex.Lock()
	defer hkm.mutex.Unlock()
	hkm.strictMode = strict
}

// placeholderKey never matches a real host: asking known_hosts about it
// returns the keys the file holds for the host
var placeholderKey = sync.OnceValue(func() ssh.PublicKey {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil
	}
	key, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil
	}
	return key
})

// HostKeyAlgorithms are the host key algorithms to ask address (host:port)
// for: the key types known_hosts holds for it, as OpenSSH does, so a server
// that prefers another type still presents the key we know; nil for an
// unknown host (the library's own order)
func (hkm *HostKeyManager) HostKeyAlgorithms(address string) []string {
	if hkm == nil || hkm.insecure {
		return nil
	}
	name := knownhosts.Normalize(address)
	hkm.mutex.RLock()
	cb := hkm.callback
	added := hkm.added[name]
	authorities := hkm.authorities
	hkm.mutex.RUnlock()

	var keyTypes []string
	if added != nil {
		keyTypes = append(keyTypes, added.Type())
	}
	if probe := placeholderKey(); cb != nil && probe != nil {
		var keyErr *knownhosts.KeyError
		if err := cb(address, probeAddr(address), probe); errors.As(err, &keyErr) {
			for _, want := range keyErr.Want {
				if authorities[string(want.Key.Marshal())] {
					// a host certificate's key type is not the CA's: keep the
					// library's order, which asks for certificates first
					return nil
				}
				keyTypes = append(keyTypes, want.Key.Type())
			}
		}
	}
	return hostKeyAlgorithms(keyTypes)
}

// probeAddr is the remote address knownhosts matches IP entries against
func probeAddr(address string) net.Addr {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	n, _ := strconv.Atoi(port)
	return &net.TCPAddr{IP: net.ParseIP(host), Port: n}
}

// hostKeyAlgorithms turns key types into the algorithms that present them:
// an RSA key signs with SHA-2 first
func hostKeyAlgorithms(keyTypes []string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(algos ...string) {
		for _, a := range algos {
			if !seen[a] {
				seen[a] = true
				out = append(out, a)
			}
		}
	}
	for _, t := range keyTypes {
		switch t {
		case ssh.KeyAlgoRSA:
			add(ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSA)
		case ssh.CertAlgoRSAv01:
			add(ssh.CertAlgoRSASHA512v01, ssh.CertAlgoRSASHA256v01, ssh.CertAlgoRSAv01)
		default:
			add(t)
		}
	}
	return out
}
