package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/onigirazu-cfg/onigirazu/internal/vault"
)

// newVaultCommand is ansible-vault's encrypt, decrypt, view and
// encrypt_string, with the same password options
func newVaultCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vault",
		Short: "Encrypt and decrypt Ansible Vault files and strings",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "encrypt FILE...",
		Short: "Encrypt files in place",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			secret, err := encryptionSecret()
			if err != nil {
				return err
			}
			for _, f := range args {
				data, err := os.ReadFile(f) // #nosec G304 -- the user names the file
				if err != nil {
					return err
				}
				if vault.IsEncrypted(data) {
					return fmt.Errorf("%s is already encrypted", f)
				}
				enc, err := vault.Encrypt(data, secret)
				if err != nil {
					return err
				}
				if err := writeKeepingMode(f, enc); err != nil {
					return err
				}
			}
			fmt.Fprintln(os.Stderr, "Encryption successful")
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "decrypt FILE...",
		Short: "Decrypt files in place",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			for _, f := range args {
				plain, err := openFile(f)
				if err != nil {
					return err
				}
				if err := writeKeepingMode(f, plain); err != nil {
					return err
				}
			}
			fmt.Fprintln(os.Stderr, "Decryption successful")
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "view FILE...",
		Short: "Print decrypted files",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			for _, f := range args {
				plain, err := openFile(f)
				if err != nil {
					return err
				}
				_, _ = os.Stdout.Write(plain)
			}
			return nil
		},
	})
	var name string
	encStr := &cobra.Command{
		Use:   "encrypt_string [STRING]",
		Short: "Encrypt a string for a YAML file (from the argument or stdin)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			var plain []byte
			if len(args) == 1 {
				plain = []byte(args[0])
			} else {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}
				plain = bytes.TrimRight(data, "\n")
			}
			secret, err := encryptionSecret()
			if err != nil {
				return err
			}
			enc, err := vault.Encrypt(plain, secret)
			if err != nil {
				return err
			}
			indent := "  "
			if name != "" {
				fmt.Printf("%s: !vault |\n", name)
				indent = "          "
			} else {
				fmt.Println("!vault |")
			}
			for _, line := range strings.Split(strings.TrimRight(string(enc), "\n"), "\n") {
				fmt.Println(indent + line)
			}
			return nil
		},
	}
	encStr.Flags().StringVarP(&name, "name", "n", "", "Variable name to print the value under")
	cmd.AddCommand(encStr)
	return cmd
}

func openFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- the user names the file
	if err != nil {
		return nil, err
	}
	if !vault.IsEncrypted(data) {
		return nil, fmt.Errorf("%s is not encrypted", path)
	}
	plain, err := vault.Open(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return plain, nil
}

// encryptionSecret is the first password given; without one, it asks twice
func encryptionSecret() (vault.Secret, error) {
	secrets, err := vault.CurrentSecrets()
	if err != nil {
		return vault.Secret{}, err
	}
	if len(secrets) > 0 {
		return secrets[0], nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return vault.Secret{}, fmt.Errorf("no vault password: use --vault-password-file, --vault-id or --ask-vault-pass")
	}
	first, err := promptVaultPassword("New")
	if err != nil {
		return vault.Secret{}, err
	}
	second, err := promptVaultPassword("Confirm")
	if err != nil {
		return vault.Secret{}, err
	}
	if !bytes.Equal(first, second) || len(first) == 0 {
		return vault.Secret{}, fmt.Errorf("the passwords do not match")
	}
	return vault.Secret{Label: "default", Password: first}, nil
}

// writeKeepingMode replaces a file's content and keeps its mode
func writeKeepingMode(path string, data []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, info.Mode().Perm())
}
