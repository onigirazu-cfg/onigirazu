package modules

import (
	"context"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// RpmKeyModule imports or removes a GPG key of rpm (Ansible's rpm_key): key
// (a file, an URL or a key id), state, fingerprint (checked when given)
type RpmKeyModule struct{ *BaseModule }

func NewRpmKeyModule() *RpmKeyModule { return &RpmKeyModule{BaseModule: NewBaseModule("rpm_key")} }

func (m *RpmKeyModule) GetDescription() string { return "Import or remove a GPG key of rpm" }

func (m *RpmKeyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error, result.Duration = false, msg, time.Since(start)
		return result, nil
	}
	key := getStringArg(args, "key", "")
	if key == "" {
		return fail("rpm_key requires 'key'")
	}
	state := getStringArg(args, "state", "present")
	// the key material: fetched for an URL, read for a file; a bare id is looked up
	fetch := "cat " + shellQuote(key)
	if strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://") {
		fetch = "(curl -fsSL " + shellQuote(key) + " || wget -qO- " + shellQuote(key) + ")"
	}
	isID := !strings.Contains(key, "/") && !strings.Contains(key, ".") && len(key) >= 8
	var keyID string
	if isID {
		keyID = strings.ToLower(key)
		if len(keyID) > 8 {
			keyID = keyID[len(keyID)-8:]
		}
	} else {
		out, err := runShellOnHost(ctx, host, args, fetch+" | gpg --with-colons --import-options show-only --import 2>/dev/null | awk -F: '/^pub/{print $5; exit}'")
		if err != nil || strings.TrimSpace(out) == "" {
			out, err = runShellOnHost(ctx, host, args, fetch+" | gpg --with-colons 2>/dev/null | awk -F: '/^pub/{print $5; exit}'")
		}
		if err != nil {
			return fail("rpm_key: reading the key: " + err.Error())
		}
		keyID = strings.ToLower(strings.TrimSpace(out))
		if len(keyID) > 8 {
			keyID = keyID[len(keyID)-8:]
		}
	}
	if keyID == "" {
		return fail("rpm_key: cannot tell the key id of " + key)
	}
	have, _ := runShellOnHost(ctx, host, args, "rpm -q gpg-pubkey-"+keyID+" >/dev/null 2>&1 && echo yes || echo no")
	present := strings.TrimSpace(have) == "yes"
	result.Output["key_id"] = keyID
	switch {
	case state == "present" && !present:
		if isID {
			return fail("rpm_key: a key id alone cannot be imported; give the key file or URL")
		}
		result.Changed = true
		if !inCheckMode(args) {
			if fp := getStringArg(args, "fingerprint", ""); fp != "" {
				out, _ := runShellOnHost(ctx, host, args, fetch+" | gpg --with-colons --with-fingerprint 2>/dev/null | awk -F: '/^fpr/{print $10; exit}'")
				if !strings.EqualFold(strings.ReplaceAll(fp, " ", ""), strings.TrimSpace(out)) {
					return fail("rpm_key: fingerprint mismatch")
				}
			}
			if _, err := runShellOnHost(ctx, host, args, fetch+" > /tmp/.onigirazu-rpm-key && rpm --import /tmp/.onigirazu-rpm-key; rc=$?; rm -f /tmp/.onigirazu-rpm-key; exit $rc"); err != nil {
				return fail("rpm_key: import: " + err.Error())
			}
		}
	case state == "absent" && present:
		result.Changed = true
		if !inCheckMode(args) {
			if _, err := runShellOnHost(ctx, host, args, "rpm -e --allmatches gpg-pubkey-"+keyID); err != nil {
				return fail("rpm_key: remove: " + err.Error())
			}
		}
	}
	result.Duration = time.Since(start)
	return result, nil
}
