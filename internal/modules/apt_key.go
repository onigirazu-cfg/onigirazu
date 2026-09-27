package modules

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// AptKeyModule installs an APT signing key as a keyring file (Ansible's
// apt_key without apt-key, which newer releases drop). The key comes from
// url, data or file; it goes to keyring, or to /etc/apt/trusted.gpg.d.
// An ASCII-armored key is written as .asc as is; a binary key file needs no
// conversion either, and an armored key for a .gpg keyring is dearmored
// with gpg on the host.
type AptKeyModule struct {
	*BaseModule
}

// NewAptKeyModule creates a new apt_key module
func NewAptKeyModule() *AptKeyModule {
	return &AptKeyModule{BaseModule: NewBaseModule("apt_key")}
}

func (m *AptKeyModule) GetDescription() string {
	return "Add or remove an APT signing key"
}

var keyID = regexp.MustCompile(`^(0x)?[0-9A-Fa-f]{8,40}$`)

func (m *AptKeyModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}
	if _, ok := args["keyserver"]; ok {
		return fmt.Errorf("apt_key: keyserver is not supported; give the key with url or data")
	}
	state := getStringArg(args, "state", "present")
	id := getStringArg(args, "id", "")
	if id != "" && !keyID.MatchString(id) {
		return fmt.Errorf("apt_key: id %q is not a key id", id)
	}
	if kr := getStringArg(args, "keyring", ""); kr != "" && !filepath.IsAbs(kr) {
		return fmt.Errorf("apt_key: keyring must be an absolute path")
	}
	sources := 0
	for _, k := range []string{"url", "data", "file"} {
		if getStringArg(args, k, "") != "" {
			sources++
		}
	}
	switch state {
	case "present":
		if sources != 1 {
			return fmt.Errorf("apt_key: give the key with exactly one of url, data or file")
		}
	case "absent":
		if getStringArg(args, "keyring", "") == "" && id == "" {
			return fmt.Errorf("apt_key: state absent needs keyring or id")
		}
	default:
		return fmt.Errorf("apt_key: state must be present or absent")
	}
	return nil
}

func armored(key []byte) bool {
	return bytes.Contains(key, []byte("-----BEGIN PGP PUBLIC KEY BLOCK-----"))
}

func (m *AptKeyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{
		TaskName: taskName(args), Host: host.Name, Module: m.name,
		Timestamp: start, Success: true, Output: map[string]interface{}{},
	}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success = false
		result.Error = msg
		result.Duration = time.Since(start)
		return result, nil
	}
	done := func() (types.TaskResult, error) {
		result.Duration = time.Since(start)
		return result, nil
	}
	state := getStringArg(args, "state", "present")
	keyring := getStringArg(args, "keyring", "")
	id := strings.ToUpper(strings.TrimPrefix(getStringArg(args, "id", ""), "0x"))

	if state == "absent" {
		paths := []string{keyring}
		if keyring == "" {
			paths = []string{"/etc/apt/trusted.gpg.d/" + id + ".asc", "/etc/apt/trusted.gpg.d/" + id + ".gpg"}
		}
		for _, p := range paths {
			if _, err := runOnHost(ctx, host, args, "test", "-e", p); err != nil {
				continue
			}
			result.Changed = true
			if !inCheckMode(args) {
				if _, err := runOnHost(ctx, host, args, "rm", "-f", p); err != nil {
					return fail(fmt.Sprintf("failed to remove %s: %v", p, err))
				}
			}
		}
		result.Output["msg"] = "key is absent"
		return done()
	}

	key, err := m.keyMaterial(ctx, host, args)
	if err != nil {
		return fail(err.Error())
	}
	if !armored(key) && !bytes.HasPrefix(key, []byte{0x99}) && !bytes.HasPrefix(key, []byte{0x98}) && !bytes.HasPrefix(key, []byte{0xc6}) {
		return fail("the key is neither an ASCII-armored nor a binary OpenPGP public key")
	}
	target := keyring
	if target == "" {
		name := id
		if name == "" {
			sum := sha256.Sum256(key)
			name = fmt.Sprintf("onigirazu-%x", sum[:6])
		}
		ext := ".gpg"
		if armored(key) {
			ext = ".asc"
		}
		target = "/etc/apt/trusted.gpg.d/" + name + ext
	}
	if armored(key) && !strings.HasSuffix(target, ".asc") {
		// a .gpg keyring holds the binary key
		if _, err := runShellOnHost(ctx, host, args, "command -v gpg"); err != nil {
			return fail(fmt.Sprintf("gpg is needed on %s to dearmor the key for %s; use a .asc keyring", host.Name, target))
		}
		out, err := runShellOnHost(ctx, host, args, fmt.Sprintf(
			"printf '%%s' %s | base64 -d | gpg --dearmor | base64 | tr -d '\\n'",
			shellQuote(base64.StdEncoding.EncodeToString(key))))
		if err != nil {
			return fail(fmt.Sprintf("failed to dearmor the key: %v", err))
		}
		if key, err = base64.StdEncoding.DecodeString(strings.TrimSpace(out)); err != nil {
			return fail(fmt.Sprintf("failed to dearmor the key: %v", err))
		}
	}
	result.Output["keyring"] = target

	current, _, err := readHostFile(ctx, host, args, target)
	if err != nil {
		return fail(err.Error())
	}
	if bytes.Equal(current, key) {
		result.Output["msg"] = "key is present"
		return done()
	}
	result.Changed = true
	if inCheckMode(args) {
		result.Output["msg"] = "would install the key in " + target
		return done()
	}
	if err := writeHostFile(ctx, host, args, target, key, 0644); err != nil {
		return fail(err.Error())
	}
	result.Output["msg"] = "key installed in " + target
	return done()
}

// keyMaterial reads the key from data, a file on the host, or a URL fetched
// by the host
func (m *AptKeyModule) keyMaterial(ctx context.Context, host types.Host, args map[string]interface{}) ([]byte, error) {
	if data := getStringArg(args, "data", ""); data != "" {
		return []byte(data), nil
	}
	if file := getStringArg(args, "file", ""); file != "" {
		key, exists, err := readHostFile(ctx, host, args, file)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("key file %s not found on %s", file, host.Name)
		}
		return key, nil
	}
	url := getStringArg(args, "url", "")
	q := shellQuote(url)
	out, err := runShellOnHost(ctx, host, args, fmt.Sprintf(
		"if command -v curl >/dev/null; then curl -fsSL %s; else wget -qO- %s; fi | base64 | tr -d '\\n'", q, q))
	if err != nil {
		return nil, fmt.Errorf("failed to download the key from %s: %v", url, err)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(out))
	if err != nil || len(key) == 0 {
		return nil, fmt.Errorf("failed to download the key from %s", url)
	}
	return key, nil
}
