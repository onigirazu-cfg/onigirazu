package expression

import (
	"bufio"
	"crypto/rand"
	"encoding/csv"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/vault"
)

// The lookups of ansible-core 2.21 beyond the core set: vars, varnames,
// inventory_hostnames, sequence, csvfile, ini, password, url, unvault, config.

func lookupVarsPlugin(vars map[string]interface{}, terms []interface{}) ([]interface{}, error) {
	names, kw := lookupArgs(terms, "default")
	out := make([]interface{}, 0, len(names))
	for _, n := range names {
		v, ok := vars[fmt.Sprint(n)]
		if !ok {
			if d, has := kw["default"]; has {
				out = append(out, d)
				continue
			}
			return nil, fmt.Errorf("lookup vars: no variable %q", n)
		}
		out = append(out, v)
	}
	return out, nil
}

func lookupVarnames(vars map[string]interface{}, terms []interface{}) ([]interface{}, error) {
	names := make([]string, 0, len(vars))
	for n := range vars {
		names = append(names, n)
	}
	sort.Strings(names)
	out := []interface{}{}
	for _, t := range terms {
		re, err := regexp.Compile(fmt.Sprint(t))
		if err != nil {
			return nil, fmt.Errorf("lookup varnames: %w", err)
		}
		for _, n := range names {
			if strings.HasPrefix(n, "__") {
				continue
			}
			if re.MatchString(n) {
				out = append(out, n)
			}
		}
	}
	return out, nil
}

// lookupInventoryHostnames expands a host pattern with the groups variable:
// names, group names, "all", a:b (union), a:!b (difference), a:&b (intersection)
func lookupInventoryHostnames(vars map[string]interface{}, terms []interface{}) ([]interface{}, error) {
	groups, _ := vars["groups"].(map[string]interface{})
	all := []string{}
	if a, ok := groups["all"]; ok {
		for _, h := range toList(a) {
			all = append(all, fmt.Sprint(h))
		}
	}
	members := func(name string) []string {
		if g, ok := groups[name]; ok {
			var out []string
			for _, h := range toList(g) {
				out = append(out, fmt.Sprint(h))
			}
			return out
		}
		for _, h := range all {
			if h == name {
				return []string{h}
			}
		}
		if strings.ContainsAny(name, "*?[") {
			var out []string
			for _, h := range all {
				if ok, _ := filepath.Match(name, h); ok {
					out = append(out, h)
				}
			}
			return out
		}
		return nil
	}
	out := []interface{}{}
	for _, t := range terms {
		result := map[string]bool{}
		var order []string
		for i, part := range strings.Split(fmt.Sprint(t), ":") {
			part = strings.TrimSpace(part)
			switch {
			case strings.HasPrefix(part, "!"):
				for _, h := range members(part[1:]) {
					delete(result, h)
				}
			case strings.HasPrefix(part, "&"):
				keep := map[string]bool{}
				for _, h := range members(part[1:]) {
					keep[h] = true
				}
				for h := range result {
					if !keep[h] {
						delete(result, h)
					}
				}
			default:
				for _, h := range members(part) {
					if !result[h] {
						order = append(order, h)
					}
					result[h] = true
				}
				_ = i
			}
		}
		for _, h := range order {
			if result[h] {
				out = append(out, h)
			}
		}
	}
	return out, nil
}

// lookupSequence: "start=1 end=5 stride=1 format=host%02d" or "1-5/2"
func lookupSequence(terms []interface{}) ([]interface{}, error) {
	out := []interface{}{}
	for _, t := range terms {
		spec := strings.TrimSpace(fmt.Sprint(t))
		start, end, stride, count := 1, 0, 1, -1
		format := "%d"
		if m := regexp.MustCompile(`^(\d+)(?:-(\d+))?(?:/(\d+))?(?::(.+))?$`).FindStringSubmatch(spec); m != nil {
			start, _ = strconv.Atoi(m[1])
			if m[2] != "" {
				end, _ = strconv.Atoi(m[2])
			} else {
				end = start
				start = 0
			}
			if m[3] != "" {
				stride, _ = strconv.Atoi(m[3])
			}
			if m[4] != "" {
				format = m[4]
			}
		} else {
			for _, f := range strings.Fields(spec) {
				k, v, _ := strings.Cut(f, "=")
				switch k {
				case "start":
					start, _ = strconv.Atoi(v)
				case "end":
					end, _ = strconv.Atoi(v)
				case "stride":
					stride, _ = strconv.Atoi(v)
				case "count":
					count, _ = strconv.Atoi(v)
				case "format":
					format = v
				}
			}
			if count >= 0 {
				end = start + (count-1)*stride
			}
		}
		if stride == 0 {
			return nil, fmt.Errorf("lookup sequence: stride 0")
		}
		if !strings.Contains(format, "%") {
			format += "%d"
		}
		for i := start; (stride > 0 && i <= end) || (stride < 0 && i >= end); i += stride {
			out = append(out, fmt.Sprintf(format, i))
		}
	}
	return out, nil
}

// lookupCSVFile: lookup('csvfile', 'key', file=..., delimiter=',', col=1, default=...)
func lookupCSVFile(resolve func(string) string, terms []interface{}) ([]interface{}, error) {
	keys, kw := lookupArgs(terms, "file", "delimiter", "col", "default", "encoding")
	file := PyStr(kw["file"])
	if file == "" {
		file = "ansible.csv"
	}
	delim := ","
	if d, ok := kw["delimiter"].(string); ok && d != "" {
		switch d {
		case "TAB", "\\t":
			delim = "\t"
		default:
			delim = d
		}
	}
	col := 1
	if c, ok := toFloat(kw["col"]); ok {
		col = int(c)
	}
	f, err := os.Open(resolve(file)) // #nosec G304 -- the playbook names the file
	if err != nil {
		return nil, fmt.Errorf("lookup csvfile: %w", err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = rune(delim[0])
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("lookup csvfile: %w", err)
	}
	out := make([]interface{}, 0, len(keys))
	for _, k := range keys {
		found := false
		for _, row := range rows {
			if len(row) > 0 && row[0] == fmt.Sprint(k) {
				if col < len(row) {
					out = append(out, row[col])
				} else {
					out = append(out, nil)
				}
				found = true
				break
			}
		}
		if !found {
			if d, ok := kw["default"]; ok {
				out = append(out, d)
			} else {
				out = append(out, nil)
			}
		}
	}
	return out, nil
}

// lookupINI: lookup('ini', 'key', file=..., section=..., type=ini|properties, re=True, default=...)
func lookupINI(resolve func(string) string, terms []interface{}) ([]interface{}, error) {
	keys, kw := lookupArgs(terms, "file", "section", "type", "re", "default", "encoding", "allow_no_value")
	file := PyStr(kw["file"])
	if file == "" {
		file = "ansible.ini"
	}
	data, err := os.ReadFile(resolve(file)) // #nosec G304 -- the playbook names the file
	if err != nil {
		return nil, fmt.Errorf("lookup ini: %w", err)
	}
	wantSection := "global"
	if s, ok := kw["section"].(string); ok && s != "" {
		wantSection = s
	}
	properties := PyStr(kw["type"]) == "properties"
	useRe := Truthy(kw["re"])
	values := map[string]string{}
	var order []string
	section := "global"
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, ";") {
			continue
		}
		if !properties && strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			section = strings.TrimSpace(t[1 : len(t)-1])
			continue
		}
		if properties || section == wantSection {
			k, v, _ := strings.Cut(t, "=")
			if !strings.Contains(t, "=") {
				k, v, _ = strings.Cut(t, ":")
			}
			k = strings.TrimSpace(k)
			values[k] = strings.TrimSpace(v)
			order = append(order, k)
		}
	}
	out := []interface{}{}
	for _, key := range keys {
		ks := fmt.Sprint(key)
		if useRe {
			re, err := regexp.Compile(ks)
			if err != nil {
				return nil, fmt.Errorf("lookup ini: %w", err)
			}
			for _, k := range order {
				if re.MatchString(k) {
					out = append(out, values[k])
				}
			}
			continue
		}
		if v, ok := values[ks]; ok {
			out = append(out, v)
		} else if d, ok := kw["default"]; ok {
			out = append(out, d)
		} else {
			out = append(out, nil)
		}
	}
	return out, nil
}

// lookupStoredRandomString is lookup('password', ...): lookup('password', '/path/to/file length=16 chars=ascii_letters,digits')
// reads the stored password or makes and stores one; /dev/null makes one without storing
func lookupStoredRandomString(resolve func(string) string, terms []interface{}) ([]interface{}, error) {
	pos, kw := lookupArgs(terms, "length", "chars", "seed", "encrypt", "ident")
	out := make([]interface{}, 0, len(pos))
	for _, t := range pos {
		spec := strings.Fields(fmt.Sprint(t))
		if len(spec) == 0 {
			return nil, fmt.Errorf("the stored-string lookup needs a path")
		}
		path := spec[0]
		length := 20
		if n, ok := toFloat(kw["length"]); ok {
			length = int(n)
		}
		chars := "ascii_letters,digits,punctuation"
		if c, ok := kw["chars"].(string); ok && c != "" {
			chars = c
		}
		for _, f := range spec[1:] {
			k, v, _ := strings.Cut(f, "=")
			switch k {
			case "length":
				length, _ = strconv.Atoi(v)
			case "chars":
				chars = v
			}
		}
		if path != "/dev/null" {
			p := resolve(path)
			if data, err := os.ReadFile(p); err == nil { // #nosec G304 -- the playbook names the file
				out = append(out, strings.TrimSpace(strings.Split(string(data), "\n")[0]))
				continue
			}
			made, err := randomStringFrom(length, chars)
			if err != nil {
				return nil, err
			}
			if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
				return nil, fmt.Errorf("stored-string lookup: %w", err)
			}
			if err := os.WriteFile(p, []byte(made+"\n"), 0o600); err != nil {
				return nil, fmt.Errorf("stored-string lookup: %w", err)
			}
			out = append(out, made)
			continue
		}
		made, err := randomStringFrom(length, chars)
		if err != nil {
			return nil, err
		}
		out = append(out, made)
	}
	return out, nil
}

func randomStringFrom(length int, chars string) (string, error) {
	sets := map[string]string{"ascii_letters": "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", "ascii_lowercase": "abcdefghijklmnopqrstuvwxyz",
		"ascii_uppercase": "ABCDEFGHIJKLMNOPQRSTUVWXYZ", "digits": "0123456789", "hexdigits": "0123456789abcdefABCDEF",
		"punctuation": "!#$%&()*+,-./:;<=>?@[]^_{|}~", "alphanumeric": "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"}
	alphabet := ""
	for _, part := range strings.Split(chars, ",") {
		part = strings.TrimSpace(part)
		if s, ok := sets[part]; ok {
			alphabet += s
		} else {
			alphabet += part
		}
	}
	if alphabet == "" || length <= 0 {
		return "", fmt.Errorf("stored-string lookup: nothing to pick from")
	}
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		b[i] = alphabet[n.Int64()]
	}
	return string(b), nil
}

// lookupURL: the body of each URL, one item per line (split_lines=True) or whole
func lookupURL(terms []interface{}) ([]interface{}, error) {
	urls, kw := lookupArgs(terms, "split_lines", "validate_certs", "timeout", "headers", "username", "password")
	timeout := 10 * time.Second
	if n, ok := toFloat(kw["timeout"]); ok {
		timeout = time.Duration(n) * time.Second
	}
	client := &http.Client{Timeout: timeout}
	out := []interface{}{}
	for _, u := range urls {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprint(u), nil)
		if err != nil {
			return nil, fmt.Errorf("lookup url: %w", err)
		}
		if h, ok := kw["headers"].(map[string]interface{}); ok {
			for k, v := range h {
				req.Header.Set(k, fmt.Sprint(v))
			}
		}
		if user, ok := kw["username"].(string); ok && user != "" {
			req.SetBasicAuth(user, PyStr(kw["password"]))
		}
		res, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("lookup url: %w", err)
		}
		body, _ := io.ReadAll(io.LimitReader(res.Body, 16<<20))
		_ = res.Body.Close()
		if res.StatusCode >= 400 {
			return nil, fmt.Errorf("lookup url: %s: %s", u, res.Status)
		}
		if v, set := kw["split_lines"]; set && !Truthy(v) {
			out = append(out, string(body))
			continue
		}
		sc := bufio.NewScanner(strings.NewReader(string(body)))
		sc.Buffer(make([]byte, 1<<20), 16<<20)
		for sc.Scan() {
			out = append(out, sc.Text())
		}
	}
	return out, nil
}

func lookupUnvault(resolve func(string) string, terms []interface{}) ([]interface{}, error) {
	out := make([]interface{}, 0, len(terms))
	for _, t := range terms {
		data, err := os.ReadFile(resolve(fmt.Sprint(t))) // #nosec G304 -- the playbook names the file
		if err != nil {
			return nil, fmt.Errorf("lookup unvault: %w", err)
		}
		if vault.IsEncrypted(data) {
			if data, err = vault.Open(data); err != nil {
				return nil, fmt.Errorf("lookup unvault: %w", err)
			}
		}
		out = append(out, string(data))
	}
	return out, nil
}

// lookupConfig answers the Ansible settings onigirazu has an equivalent for
func lookupConfig(terms []interface{}) ([]interface{}, error) {
	settings := map[string]interface{}{
		"DEFAULT_REMOTE_TMP": "~/.onigirazu/tmp", "DEFAULT_FORKS": 10, "DEFAULT_TIMEOUT": 30, "DEFAULT_GATHERING": "implicit",
		"DEFAULT_HOST_LIST": []interface{}{"/etc/ansible/hosts"}, "DEFAULT_ROLES_PATH": []interface{}{"roles"},
		"DEFAULT_STRATEGY": "linear", "HOST_KEY_CHECKING": true,
		"DEFAULT_BECOME": false, "DEFAULT_BECOME_METHOD": "sudo", "DEFAULT_BECOME_USER": "root",
		"RETRY_FILES_ENABLED": false, "DEFAULT_MANAGED_STR": "Ansible managed", "ANSIBLE_PIPELINING": true,
	}
	names, kw := lookupArgs(terms, "plugin_type", "plugin_name", "on_missing")
	out := make([]interface{}, 0, len(names))
	for _, n := range names {
		v, ok := settings[strings.ToUpper(fmt.Sprint(n))]
		if !ok {
			if PyStr(kw["on_missing"]) == "skip" {
				continue
			}
			if PyStr(kw["on_missing"]) == "warn" {
				out = append(out, nil)
				continue
			}
			return nil, fmt.Errorf("lookup config: unknown setting %q", n)
		}
		out = append(out, v)
	}
	return out, nil
}
