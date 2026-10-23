package cli

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// allTokens also supports configs constructed without loadServeConfig.
func (a ServeAuth) allTokens() []ServeToken {
	if a.Token == "" || slices.ContainsFunc(a.Tokens, func(t ServeToken) bool { return strings.EqualFold(t.Name, "token") }) {
		return a.Tokens
	}
	return append(slices.Clone(a.Tokens), ServeToken{Name: "token", Token: a.Token})
}

func parseServeTokenExpiry(value string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, value)
}

func (t *ServeToken) prepare() error {
	if t.Role == "" {
		t.Role = "operator"
	}
	if t.Role != "operator" && t.Role != "viewer" {
		return fmt.Errorf("auth.tokens: %q: role must be operator or viewer", t.Name)
	}
	if t.Expires != "" {
		expiry, err := parseServeTokenExpiry(t.Expires)
		if err != nil {
			return fmt.Errorf("auth.tokens: %q: invalid expires: %w", t.Name, err)
		}
		t.expires = expiry
	}
	if strings.HasPrefix(t.Token, "sha256:") {
		digest, err := hex.DecodeString(strings.TrimPrefix(t.Token, "sha256:"))
		if err != nil || len(digest) != sha256.Size {
			return fmt.Errorf("auth.tokens: %q: sha256 token must contain 64 hex digits", t.Name)
		}
		copy(t.digest[:], digest)
	} else {
		t.digest = sha256.Sum256([]byte(t.Token))
	}
	t.prepared = true
	return nil
}

type serveTokenOptions struct {
	name, role, expires string
	jobs, groups        []string
}

func newServeTokenCmd() *cobra.Command {
	o := &serveTokenOptions{}
	cmd := &cobra.Command{Use: "token", Short: "Generate bearer tokens for serve"}
	newCmd := &cobra.Command{
		Use: "new", Short: "Print a new token and its hashed YAML entry", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			value, snippet, err := generateServeToken(*o, rand.Reader, time.Now)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Token for %s (shown once, store it in your secret manager):\n%s\n\nAdd to auth.tokens in serve.yml:\n%s", o.name, value, snippet)
			return err
		},
	}
	newCmd.Flags().StringVar(&o.name, "name", "", "Token name (lowercase letters, digits, dots, underscores and hyphens; max 64)")
	_ = newCmd.MarkFlagRequired("name")
	newCmd.Flags().StringVar(&o.role, "role", "operator", "Token role: operator or viewer")
	newCmd.Flags().StringSliceVar(&o.jobs, "jobs", nil, "Jobs this token may operate (comma-separated)")
	newCmd.Flags().StringSliceVar(&o.groups, "groups", nil, "Token groups (comma-separated)")
	newCmd.Flags().StringVar(&o.expires, "expires", "90d", "Expiry: days (90d), date, RFC3339 or never")
	cmd.AddCommand(newCmd)
	return cmd
}

var serveTokenName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
var serveTokenDays = regexp.MustCompile(`^[0-9]+d$`)

func generateServeToken(o serveTokenOptions, random io.Reader, now func() time.Time) (string, string, error) {
	if !serveTokenName.MatchString(o.name) {
		return "", "", fmt.Errorf("name must match %s", serveTokenName)
	}
	t := ServeToken{Name: o.name, Role: o.role, Jobs: o.jobs, Groups: o.groups}
	if t.Role == "" {
		t.Role = "operator"
	}
	if o.expires == "" {
		o.expires = "90d"
	}
	switch {
	case o.expires == "never":
	case serveTokenDays.MatchString(o.expires):
		days, err := strconv.Atoi(strings.TrimSuffix(o.expires, "d"))
		if err != nil || days > 3652058 {
			return "", "", fmt.Errorf("expires: invalid number of days")
		}
		expiry := now().UTC().AddDate(0, 0, days)
		if expiry.Year() > 9999 {
			return "", "", fmt.Errorf("expires: date exceeds year 9999")
		}
		t.Expires = expiry.Format("2006-01-02")
	default:
		if _, err := parseServeTokenExpiry(o.expires); err != nil {
			return "", "", fmt.Errorf("expires: %w", err)
		}
		t.Expires = o.expires
	}
	if err := t.prepare(); err != nil {
		return "", "", err
	}
	var raw [32]byte
	if _, err := io.ReadFull(random, raw[:]); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	value := "ozs_" + base64.RawURLEncoding.EncodeToString(raw[:])
	digest := sha256.Sum256([]byte(value))
	t.Token = "sha256:" + hex.EncodeToString(digest[:])
	snippet, err := serveTokenSnippet(t)
	return value, snippet, err
}

func serveTokenSnippet(t ServeToken) (string, error) {
	// A flow list keeps jobs and groups compact; YAML quotes unsafe names.
	data, err := yaml.Marshal([]ServeToken{t})
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		b.WriteString("  " + line + "\n")
	}
	return b.String(), nil
}
