package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/onigirazu-cfg/onigirazu/internal/vault"
)

var (
	vaultPasswordFiles []string
	vaultIDs           []string
	askVaultPass       bool
)

// addVaultFlags adds the vault password flags to the root command
func addVaultFlags(rootCmd *cobra.Command) {
	flags := rootCmd.PersistentFlags()
	flags.StringArrayVar(&vaultPasswordFiles, "vault-password-file", nil, "File with the Ansible Vault password, or a script that prints it (repeatable; default: $ANSIBLE_VAULT_PASSWORD_FILE, ansible.cfg vault_password_file)")
	flags.StringArrayVar(&vaultIDs, "vault-id", nil, "Vault id as label@file or label@prompt (repeatable; default: $ANSIBLE_VAULT_IDENTITY_LIST, ansible.cfg vault_identity_list)")
	flags.BoolVarP(&askVaultPass, "ask-vault-pass", "J", false, "Ask for the Ansible Vault password")
	cobra.OnInitialize(func() { vault.SetSource(vaultSecrets) })
}

// vaultSecrets collects the vault passwords from the flags, then Ansible's
// environment variables and ansible.cfg, as ansible-playbook does
func vaultSecrets() ([]vault.Secret, error) {
	ids := append([]string{}, vaultIDs...)
	files := append([]string{}, vaultPasswordFiles...)
	if env := os.Getenv("ANSIBLE_VAULT_IDENTITY_LIST"); env != "" {
		ids = append(ids, splitList(env)...)
	}
	if env := os.Getenv("ANSIBLE_VAULT_PASSWORD_FILE"); env != "" {
		files = append(files, splitList(env)...)
	}
	cfgIDs, cfgFile := ansibleCfgVault()
	ids = append(ids, cfgIDs...)
	if cfgFile != "" {
		files = append(files, cfgFile)
	}

	var out []vault.Secret
	add := func(label, source string) error {
		if label == "" {
			label = "default"
		}
		var pw []byte
		var err error
		if source == "prompt" {
			pw, err = promptVaultPassword(label)
		} else {
			pw, err = readVaultPasswordFile(source)
		}
		if err != nil {
			return err
		}
		out = append(out, vault.Secret{Label: label, Password: pw})
		return nil
	}
	for _, id := range ids {
		label, source, ok := strings.Cut(id, "@")
		if !ok {
			label, source = "default", id
		}
		if err := add(label, source); err != nil {
			return nil, err
		}
	}
	for _, f := range files {
		if err := add("default", f); err != nil {
			return nil, err
		}
	}
	if askVaultPass {
		if err := add("default", "prompt"); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// readVaultPasswordFile reads a password file, or runs it when it is
// executable and takes its output
func readVaultPasswordFile(path string) ([]byte, error) {
	path = expandHome(path)
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("vault password file: %w", err)
	}
	var data []byte
	if info.Mode()&0o111 != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, path) // #nosec G204 -- the user's own password script
		cmd.Stderr = os.Stderr
		if data, err = cmd.Output(); err != nil {
			return nil, fmt.Errorf("vault password script %s: %w", path, err)
		}
	} else if data, err = os.ReadFile(path); err != nil { // #nosec G304 -- the user names the file
		return nil, fmt.Errorf("vault password file: %w", err)
	}
	pw := bytes.TrimRight(data, "\r\n")
	if len(pw) == 0 {
		return nil, fmt.Errorf("vault password file %s is empty", path)
	}
	return pw, nil
}

func promptVaultPassword(label string) ([]byte, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, fmt.Errorf("--ask-vault-pass needs a terminal; use --vault-password-file")
	}
	prompt := "Vault password: "
	if label != "default" {
		prompt = fmt.Sprintf("Vault password (%s): ", label)
	}
	fmt.Fprint(os.Stderr, prompt)
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return pw, err
}

// ansibleCfgVault reads vault_identity_list and vault_password_file from the
// [defaults] of the ansible.cfg Ansible would use
func ansibleCfgVault() (ids []string, passwordFile string) {
	candidates := []string{}
	if env := os.Getenv("ANSIBLE_CONFIG"); env != "" {
		candidates = append(candidates, env)
	}
	candidates = append(candidates, "ansible.cfg")
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".ansible.cfg"))
	}
	candidates = append(candidates, "/etc/ansible/ansible.cfg")
	for _, c := range candidates {
		f, err := os.Open(c) // #nosec G304 -- Ansible's config file
		if err != nil {
			continue
		}
		base := filepath.Dir(c)
		section := ""
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if strings.HasPrefix(line, "[") {
				section = strings.Trim(line, "[] ")
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok || section != "defaults" {
				continue
			}
			value = strings.TrimSpace(value)
			switch strings.TrimSpace(key) {
			case "vault_password_file":
				passwordFile = relativeTo(base, expandHome(value))
			case "vault_identity_list":
				for _, id := range splitList(value) {
					if label, src, ok := strings.Cut(id, "@"); ok && src != "prompt" {
						id = label + "@" + relativeTo(base, expandHome(src))
					}
					ids = append(ids, id)
				}
			}
		}
		_ = f.Close()
		return ids, passwordFile
	}
	return nil, ""
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

func relativeTo(base, p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}
