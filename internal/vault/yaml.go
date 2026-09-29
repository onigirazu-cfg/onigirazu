package vault

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

var (
	mu      sync.Mutex
	secrets []Secret
	source  func() ([]Secret, error)
	loaded  bool
	loadErr error
)

// SetSecrets sets the passwords that open vault data
func SetSecrets(s []Secret) {
	mu.Lock()
	secrets, source, loaded, loadErr = s, nil, true, nil
	mu.Unlock()
}

// SetSource sets where the passwords come from; it is called the first time
// vault data is met, so a prompt appears only when a password is needed
func SetSource(f func() ([]Secret, error)) {
	mu.Lock()
	secrets, source, loaded, loadErr = nil, f, false, nil
	mu.Unlock()
}

// CurrentSecrets returns the passwords, asking the source once
func CurrentSecrets() ([]Secret, error) {
	mu.Lock()
	defer mu.Unlock()
	if !loaded && source != nil {
		secrets, loadErr = source()
		loaded = true
	}
	return secrets, loadErr
}

// Open decrypts vault data with the current passwords
func Open(data []byte) ([]byte, error) {
	s, err := CurrentSecrets()
	if err != nil {
		return nil, err
	}
	return Decrypt(data, s)
}

// ReadFile reads a YAML file (playbook, vars, inventory, role file) and
// opens its vault data: the whole file, or !vault values inside it
func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- callers read the playbook's own files
	if err != nil {
		return nil, err
	}
	out, err := Load(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return out, nil
}

// Load opens the vault data of YAML text: a whole encrypted file becomes its
// plain text, a !vault value its plain string; other text is returned as is
func Load(data []byte) ([]byte, error) {
	if IsEncrypted(data) {
		plain, err := Open(data)
		if err != nil {
			return nil, err
		}
		data = plain
	}
	if !bytes.Contains(data, []byte("!vault")) {
		return data, nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return data, nil // not YAML the tag could be in; the caller reports parse errors
	}
	changed := false
	if err := openVaultNodes(&doc, &changed); err != nil {
		return nil, err
	}
	if !changed {
		return data, nil
	}
	return yaml.Marshal(&doc)
}

func openVaultNodes(n *yaml.Node, changed *bool) error {
	if n.Kind == yaml.ScalarNode && n.Tag == "!vault" {
		plain, err := Open([]byte(strings.TrimSpace(n.Value)))
		if err != nil {
			return fmt.Errorf("line %d: %w", n.Line, err)
		}
		n.Tag, n.Value, n.Style = "!!str", string(plain), yaml.DoubleQuotedStyle
		*changed = true
		return nil
	}
	for _, c := range n.Content {
		if err := openVaultNodes(c, changed); err != nil {
			return err
		}
	}
	return nil
}
