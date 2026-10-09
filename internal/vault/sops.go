package vault

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

// SOPS files (YAML or JSON encrypted by Mozilla SOPS) carry a top-level
// "sops" mapping with the key metadata and a MAC. They are opened with the
// sops binary, which holds the keys (age, PGP, KMS); the plain text is
// kept for the run so a file is decrypted once.

var sopsMeta = regexp.MustCompile(`(?m)(?:^sops\s*:|"sops"\s*:)`)

// IsSOPS reports a YAML or JSON document encrypted by SOPS
func IsSOPS(data []byte) bool {
	return sopsMeta.Match(data) && bytes.Contains(data, []byte("mac")) && bytes.Contains(data, []byte("ENC["))
}

var (
	sopsMu    sync.Mutex
	sopsPlain = map[[32]byte][]byte{}
	// sopsCommand runs sops; tests replace it
	sopsCommand = func(args ...string) *exec.Cmd { return exec.Command("sops", args...) } // #nosec G204 -- fixed arguments
)

// DecryptSOPS opens a SOPS document through the sops binary; the format
// is kept (YAML stays YAML, JSON stays JSON)
func DecryptSOPS(data []byte) ([]byte, error) {
	key := sha256.Sum256(data)
	sopsMu.Lock()
	plain, ok := sopsPlain[key]
	sopsMu.Unlock()
	if ok {
		return plain, nil
	}
	format := "yaml"
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
		format = "json"
	}
	cmd := sopsCommand("--decrypt", "--input-type", format, "--output-type", format, "/dev/stdin")
	cmd.Stdin = bytes.NewReader(data)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, errors.New("sops: the file is encrypted with SOPS but sops is not installed")
		}
		return nil, fmt.Errorf("sops: %s", strings.TrimSpace(firstLine(stderr.String(), err.Error())))
	}
	sopsMu.Lock()
	sopsPlain[key] = out
	sopsMu.Unlock()
	return out, nil
}

func firstLine(s, fallback string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
