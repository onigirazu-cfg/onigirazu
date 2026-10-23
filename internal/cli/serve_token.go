package cli

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
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
	var configFile string
	var printOnly bool
	cmd := &cobra.Command{Use: "token", Short: "Generate bearer tokens for serve"}
	newCmd := &cobra.Command{
		Use: "new", Short: "Generate and save a new token", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !printOnly {
				cfg, err := loadServeConfig(configFile)
				if err != nil {
					return err
				}
				if err = validateServeTokenJobs(cfg, o.jobs); err != nil {
					return err
				}
				current, err := user.Current()
				if err != nil {
					return err
				}
				value, record, err := makeStoredServeToken(*o, "cli:"+current.Username, time.Now)
				if err != nil {
					return err
				}
				if _, err = addServeToken(cfg, record); err != nil {
					return err
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Token for %s (shown once, store it in your secret manager):\n%s\nSaved to %s; a running serve picks it up within seconds.\n", record.Name, value, filepath.Join(cfg.DataDir, "tokens.json"))
				return err
			}
			value, t, err := generateServeTokenValue(*o, rand.Reader, time.Now)
			if err != nil {
				return err
			}
			snippet, err := serveTokenSnippet(t)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Token for %s (shown once, store it in your secret manager):\n%s\n\nAdd to auth.tokens in serve.yml:\n%s", t.Name, value, snippet)
			return err
		},
	}
	newCmd.Flags().StringVar(&o.name, "name", "", "Token name (lowercase letters, digits, dots, underscores and hyphens; max 64); default from the role and job")
	newCmd.Flags().StringVar(&o.role, "role", "operator", "Token role: operator or viewer")
	newCmd.Flags().StringSliceVar(&o.jobs, "jobs", nil, "Jobs this token may operate (comma-separated)")
	newCmd.Flags().StringSliceVar(&o.groups, "groups", nil, "Token groups (comma-separated)")
	newCmd.Flags().StringVar(&o.expires, "expires", "90d", "Expiry: days (90d), date, RFC3339 or never")
	cmd.PersistentFlags().StringVarP(&configFile, "config-file", "f", "serve.yml", "Serve configuration file")
	newCmd.Flags().BoolVar(&printOnly, "print", false, "Print the token and hashed YAML entry without reading configuration")
	listCmd := &cobra.Command{Use: "list", Short: "List serve tokens", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := loadServeConfig(configFile)
		if err != nil {
			return err
		}
		tokens, err := readServeTokens(cfg.DataDir)
		if err != nil {
			return err
		}
		s := &server{cfg: cfg, now: time.Now}
		s.tokens.tokens = tokens
		s.tokens.usage = map[string]time.Time{}
		if data, err := os.ReadFile(filepath.Join(cfg.DataDir, "token-usage.json")); err == nil {
			if err = json.Unmarshal(data, &s.tokens.usage); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSOURCE\tROLE\tJOBS\tEXPIRES\tLAST USED")
		for _, t := range s.tokenViewsLocked() {
			expires, last := "never", "-"
			if t.ExpiresAt != nil {
				expires = t.ExpiresAt.Format(time.RFC3339)
			}
			if t.LastUsedAt != nil {
				last = t.LastUsedAt.Format(time.RFC3339)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", t.Name, t.Source, t.Role, strings.Join(t.Jobs, ","), expires, last)
		}
		return w.Flush()
	}}
	revokeCmd := &cobra.Command{Use: "revoke NAME", Short: "Revoke a stored serve token", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadServeConfig(configFile)
		if err != nil {
			return err
		}
		_, err = revokeServeToken(cfg, args[0], serveIdentity{})
		return err
	}}
	cmd.AddCommand(newCmd, listCmd, revokeCmd)
	return cmd
}

var serveTokenName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
var serveTokenDays = regexp.MustCompile(`^[0-9]+d$`)

func generateServeToken(o serveTokenOptions, random io.Reader, now func() time.Time) (string, string, error) {
	value, t, err := generateServeTokenValue(o, random, now)
	if err != nil {
		return "", "", err
	}
	snippet, err := serveTokenSnippet(t)
	return value, snippet, err
}

var serveTokenNameJunk = regexp.MustCompile(`[^a-z0-9._-]+`)

// autoServeTokenName names a token nobody named: its role, its job when it
// has one, and four random hex digits (viewer-drift-site-3f2a)
func autoServeTokenName(o serveTokenOptions, random io.Reader) (string, error) {
	base := o.role
	if base == "" {
		base = "operator"
	}
	if len(o.jobs) == 1 {
		if job := strings.Trim(serveTokenNameJunk.ReplaceAllString(strings.ToLower(o.jobs[0]), "-"), "-._"); job != "" {
			base += "-" + job
		}
	}
	if len(base) > 58 {
		base = strings.TrimRight(base[:58], "-._")
	}
	var b [2]byte
	if _, err := io.ReadFull(random, b[:]); err != nil {
		return "", fmt.Errorf("generate token name: %w", err)
	}
	return base + "-" + hex.EncodeToString(b[:]), nil
}

func generateServeTokenValue(o serveTokenOptions, random io.Reader, now func() time.Time) (string, ServeToken, error) {
	if o.name == "" {
		name, err := autoServeTokenName(o, random)
		if err != nil {
			return "", ServeToken{}, err
		}
		o.name = name
	}
	if !serveTokenName.MatchString(o.name) {
		return "", ServeToken{}, fmt.Errorf("name must match %s", serveTokenName)
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
			return "", ServeToken{}, fmt.Errorf("expires: invalid number of days")
		}
		expiry := now().UTC().AddDate(0, 0, days)
		if expiry.Year() > 9999 {
			return "", ServeToken{}, fmt.Errorf("expires: date exceeds year 9999")
		}
		t.Expires = expiry.Format("2006-01-02")
	default:
		if _, err := parseServeTokenExpiry(o.expires); err != nil {
			return "", ServeToken{}, fmt.Errorf("expires: %w", err)
		}
		t.Expires = o.expires
	}
	if err := t.prepare(); err != nil {
		return "", ServeToken{}, err
	}
	var raw [32]byte
	if _, err := io.ReadFull(random, raw[:]); err != nil {
		return "", ServeToken{}, fmt.Errorf("generate token: %w", err)
	}
	value := "ozs_" + base64.RawURLEncoding.EncodeToString(raw[:])
	digest := sha256.Sum256([]byte(value))
	t.Token = "sha256:" + hex.EncodeToString(digest[:])
	t.digest = digest
	return value, t, nil
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
