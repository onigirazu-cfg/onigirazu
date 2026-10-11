package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

// bootstrap takes a fresh machine — reachable as root (or an installer's
// user) with a password or a key — and makes it a managed host: a deploy
// user with your public key and passwordless sudo, password logins off if
// wanted. It is one playbook, generated and run here, then a check that the
// new user works.

type bootstrapOptions struct {
	user, keyFile, passwordEnv string
	askPass                    bool
	newUser, pubkey, shell     string
	sudoGroup                  string
	disablePasswordAuth        bool
	keepRootLogin              bool
	dryRun                     bool
}

func newBootstrapCmd() *cobra.Command {
	o := &bootstrapOptions{}
	cmd := &cobra.Command{
		Use:   "bootstrap HOST[:PORT]",
		Short: "Make a fresh machine a managed host: deploy user, your key, passwordless sudo",
		Long: `Connect with the initial credentials (--user, --ask-pass or --key), create the
deploy user, install your public key for it, allow it passwordless sudo, and
optionally turn password logins off. Then connect as the new user with the key
and run sudo -n true to prove it works, and print the inventory line.`,
		Example: `  onigirazu bootstrap 192.168.1.50 --ask-pass                      # as root with a password
  onigirazu bootstrap new-vm:2222 -u ubuntu --key ~/.ssh/cloud.pem --new-user deploy
  onigirazu bootstrap 10.0.0.9 --ask-pass --disable-password-auth`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBootstrap(cmd, args[0], o)
		},
	}
	f := cmd.Flags()
	f.StringVarP(&o.user, "user", "u", "root", "Initial SSH user")
	f.StringVar(&o.keyFile, "key", "", "Private key of the initial user")
	f.BoolVar(&o.askPass, "ask-pass", false, "Ask for the initial user's password")
	f.StringVar(&o.passwordEnv, "password-env", "", "Environment variable that holds the initial user's password")
	f.StringVar(&o.newUser, "new-user", "deploy", "User to create")
	f.StringVar(&o.pubkey, "pubkey", "", "Public key to install for the new user (default: ~/.ssh/id_ed25519.pub, then id_rsa.pub)")
	f.StringVar(&o.shell, "shell", "/bin/bash", "Shell of the new user")
	f.StringVar(&o.sudoGroup, "sudo-group", "", "Admin group to add the user to (default: sudo, else wheel)")
	f.BoolVar(&o.disablePasswordAuth, "disable-password-auth", false, "Set PasswordAuthentication no in sshd and reload it")
	f.BoolVar(&o.keepRootLogin, "keep-root-login", true, "Leave PermitRootLogin as it is (false: set it to prohibit-password)")
	f.BoolVar(&o.dryRun, "check", false, "Show what would change, change nothing")
	return cmd
}

// bootstrapPlaybook is the playbook bootstrap runs; pubkey is the key text
func bootstrapPlaybook(o *bootstrapOptions, pubkey string) ([]byte, error) {
	sudoers := o.newUser + " ALL=(ALL) NOPASSWD:ALL\n"
	tasks := []map[string]interface{}{
		{"name": "The admin group", "shell": "getent group sudo >/dev/null && echo sudo || { getent group wheel >/dev/null && echo wheel || echo none; }",
			"register": "admin_group", "changed_when": false},
		{"name": "The user " + o.newUser, "user": map[string]interface{}{"name": o.newUser, "shell": o.shell, "create_home": true,
			"groups": "{{ '" + o.sudoGroup + "' if '" + o.sudoGroup + "' else (admin_group.stdout if admin_group.stdout != 'none' else omit) }}", "append": true}},
		{"name": "Its authorized key", "authorized_key": map[string]interface{}{"user": o.newUser, "key": strings.TrimSpace(pubkey)}},
		{"name": "Passwordless sudo", "copy": map[string]interface{}{"dest": "/etc/sudoers.d/90-" + o.newUser, "content": sudoers, "mode": "0440", "owner": "root", "group": "root", "validate": "visudo -cf %s"}},
	}
	if o.disablePasswordAuth {
		tasks = append(tasks, map[string]interface{}{"name": "No password logins", "lineinfile": map[string]interface{}{
			"path": "/etc/ssh/sshd_config", "regexp": "^#?PasswordAuthentication ", "line": "PasswordAuthentication no", "validate": "sshd -t -f %s"}, "notify": "reload sshd"})
	}
	if !o.keepRootLogin {
		tasks = append(tasks, map[string]interface{}{"name": "Root only with a key", "lineinfile": map[string]interface{}{
			"path": "/etc/ssh/sshd_config", "regexp": "^#?PermitRootLogin ", "line": "PermitRootLogin prohibit-password", "validate": "sshd -t -f %s"}, "notify": "reload sshd"})
	}
	play := map[string]interface{}{"name": "bootstrap " + o.newUser, "hosts": "all", "become": true, "gather_facts": false, "tasks": tasks,
		"handlers": []map[string]interface{}{{"name": "reload sshd", "shell": "systemctl reload ssh 2>/dev/null || systemctl reload sshd 2>/dev/null || service ssh reload"}}}
	return yaml.Marshal(map[string]interface{}{"plays": []interface{}{play}})
}

func splitHostPort(target string) (string, int) {
	host, port := target, 22
	if i := strings.LastIndex(target, ":"); i > 0 && !strings.Contains(target[i+1:], "]") {
		if n, err := strconv.Atoi(target[i+1:]); err == nil {
			host, port = target[:i], n
		}
	}
	return strings.Trim(host, "[]"), port
}

func readPublicKey(path string) (string, string, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		for _, name := range []string{"id_ed25519.pub", "id_rsa.pub", "id_ecdsa.pub"} {
			p := filepath.Join(home, ".ssh", name)
			if data, err := os.ReadFile(p); err == nil { // #nosec G304 -- the user's own key
				return string(data), strings.TrimSuffix(p, ".pub"), nil
			}
		}
		return "", "", fmt.Errorf("no public key in ~/.ssh (--pubkey)")
	}
	data, err := os.ReadFile(path) // #nosec G304 -- the user's own key
	if err != nil {
		return "", "", err
	}
	return string(data), strings.TrimSuffix(path, ".pub"), nil
}

func runBootstrap(cmd *cobra.Command, target string, o *bootstrapOptions) error {
	out := cmd.OutOrStdout()
	host, port := splitHostPort(target)
	pubkey, privateKey, err := readPublicKey(o.pubkey)
	if err != nil {
		return err
	}
	password := ""
	switch {
	case o.passwordEnv != "":
		password = os.Getenv(o.passwordEnv)
	case o.askPass:
		fmt.Fprintf(os.Stderr, "%s@%s password: ", o.user, host)
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return err
		}
		password = string(b)
	}
	if password == "" && o.keyFile == "" {
		return fmt.Errorf("the initial credentials are needed: --ask-pass, --password-env or --key")
	}
	dir, err := os.MkdirTemp("", "onigirazu-bootstrap-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	hostVars := map[string]interface{}{"ansible_host": host, "ansible_port": port, "ansible_user": o.user,
		"ansible_ssh_common_args": "-o StrictHostKeyChecking=accept-new"}
	if password != "" {
		hostVars["ansible_password"] = password
		hostVars["ansible_become_password"] = password
	}
	if o.keyFile != "" {
		hostVars["ansible_ssh_private_key_file"] = o.keyFile
	}
	writeInv := func(name string, vars map[string]interface{}) (string, error) {
		data, err := yaml.Marshal(map[string]interface{}{"all": map[string]interface{}{"hosts": map[string]interface{}{"target": vars}}})
		if err != nil {
			return "", err
		}
		p := filepath.Join(dir, name)
		return p, os.WriteFile(p, data, 0o600)
	}
	inventory, err := writeInv("initial.yml", hostVars)
	if err != nil {
		return err
	}
	playbook, err := bootstrapPlaybook(o, pubkey)
	if err != nil {
		return err
	}
	pbPath := filepath.Join(dir, "bootstrap.yml")
	if err := os.WriteFile(pbPath, playbook, 0o600); err != nil {
		return err
	}
	inventoryPaths = []string{inventory}
	statePath = filepath.Join(dir, ".onigirazu-state")
	args := []string{pbPath}
	if o.dryRun {
		args = append(args, "--check", "--diff")
	}
	fmt.Fprintf(out, "bootstrap: %s@%s:%d → user %s with %s\n", o.user, host, port, o.newUser, strings.TrimSuffix(filepath.Base(privateKey), ".pub")+".pub")
	result, err := runPlaybook(args)
	if err != nil {
		return err
	}
	failed := 0
	for _, play := range result.Plays {
		for _, h := range play.Hosts {
			for _, t := range h.Tasks {
				if t.Failed && !t.Ignored {
					failed++
					fmt.Fprintf(out, "bootstrap: %s failed: %s\n", t.TaskName, t.Error)
				}
			}
		}
	}
	if failed > 0 {
		return &ExitError{Code: 1}
	}
	if o.dryRun {
		return nil
	}
	// the proof: the new user, the key, sudo
	check, err := writeInv("new.yml", map[string]interface{}{"ansible_host": host, "ansible_port": port, "ansible_user": o.newUser,
		"ansible_ssh_private_key_file": privateKey, "ansible_ssh_common_args": "-o StrictHostKeyChecking=accept-new"})
	if err != nil {
		return err
	}
	checkPB := filepath.Join(dir, "check.yml")
	if err := os.WriteFile(checkPB, []byte("plays:\n  - hosts: all\n    gather_facts: false\n    tasks:\n      - command: sudo -n true\n        changed_when: false\n"), 0o600); err != nil {
		return err
	}
	inventoryPaths = []string{check}
	if res, err := runPlaybook([]string{checkPB}); err != nil || imageFailed(res) {
		return fmt.Errorf("bootstrap: the new user %s cannot log in with the key and sudo: %v", o.newUser, err)
	}
	fmt.Fprintf(out, "bootstrap: done. Inventory line:\n  %s ansible_host=%s ansible_port=%d ansible_user=%s ansible_ssh_private_key_file=%s\n", host, host, port, o.newUser, privateKey)
	return nil
}
