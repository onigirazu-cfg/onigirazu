package expression

import (
	"crypto/md5"  // #nosec G501 -- the md5 filter is what Ansible offers
	"crypto/sha1" // #nosec G505 -- the sha1 filter is what Ansible offers
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// The ansible.builtin filters of ansible-core 2.21 that the core set lacked:
// hashes, paths, maths, itertools, urls, dates, type_debug, comment,
// human_readable/human_to_bytes, rekey_on_member, from_yaml_all.

var extraFilterNames = []string{"checksum", "md5", "sha1", "combinations", "permutations", "zip_longest", "commonpath",
	"comment", "expanduser", "expandvars", "fileglob", "from_yaml_all", "human_readable", "human_to_bytes", "log", "normpath",
	"path_join", "pow", "realpath", "rekey_on_member", "relpath", "root", "strftime", "subelements", "to_datetime", "to_uuid",
	"type_debug", "unvault", "urldecode", "urlsplit", "win_basename", "win_dirname", "win_splitdrive", "product"}

type filterFunc = func(params ...interface{}) (interface{}, error)

// extraFilterTable is built once, at package init: the filter list is
// assembled per evaluation and hosts evaluate in parallel, so a lazy table
// would be written from several goroutines at once
var extraFilterTable = func() map[string]filterFunc {
	t := map[string]filterFunc{}
	builtinFilters221(func(name string, f filterFunc) interface{} { t[name] = f; return nil })
	return t
}()

// extraFilter is the named filter of the 2.21 set
func extraFilter(name string) filterFunc { return extraFilterTable[name] }

func builtinFilters221(fn func(name string, f func(params ...interface{}) (interface{}, error)) interface{}) []interface{} {
	str := PyStr
	hashOf := func(h func() []byte) (interface{}, error) { return hex.EncodeToString(h()), nil }
	return []interface{}{
		fn("checksum", func(p ...interface{}) (interface{}, error) {
			return hashOf(func() []byte { s := sha1.Sum([]byte(str(p[0]))); return s[:] }) // #nosec G401 -- Ansible's checksum is sha1
		}),
		fn("sha1", func(p ...interface{}) (interface{}, error) {
			return hashOf(func() []byte { s := sha1.Sum([]byte(str(p[0]))); return s[:] }) // #nosec G401
		}),
		fn("md5", func(p ...interface{}) (interface{}, error) {
			return hashOf(func() []byte { s := md5.Sum([]byte(str(p[0]))); return s[:] }) // #nosec G401
		}),
		fn("sha256", func(p ...interface{}) (interface{}, error) {
			return hashOf(func() []byte { s := sha256.Sum256([]byte(str(p[0]))); return s[:] })
		}),
		fn("combinations", func(p ...interface{}) (interface{}, error) {
			items := toList(p[0])
			k := 2
			if len(p) > 1 {
				if n, ok := toFloat(p[1]); ok {
					k = int(n)
				}
			}
			var out []interface{}
			var rec func(start int, cur []interface{})
			rec = func(start int, cur []interface{}) {
				if len(cur) == k {
					out = append(out, append([]interface{}{}, cur...))
					return
				}
				for i := start; i < len(items); i++ {
					rec(i+1, append(cur, items[i]))
				}
			}
			rec(0, nil)
			if out == nil {
				out = []interface{}{}
			}
			return out, nil
		}),
		fn("permutations", func(p ...interface{}) (interface{}, error) {
			items := toList(p[0])
			k := len(items)
			if len(p) > 1 {
				if n, ok := toFloat(p[1]); ok {
					k = int(n)
				}
			}
			var out []interface{}
			used := make([]bool, len(items))
			var rec func(cur []interface{})
			rec = func(cur []interface{}) {
				if len(cur) == k {
					out = append(out, append([]interface{}{}, cur...))
					return
				}
				for i := range items {
					if used[i] {
						continue
					}
					used[i] = true
					rec(append(cur, items[i]))
					used[i] = false
				}
			}
			rec(nil)
			if out == nil {
				out = []interface{}{}
			}
			return out, nil
		}),
		fn("product", func(p ...interface{}) (interface{}, error) {
			lists := [][]interface{}{toList(p[0])}
			for _, x := range p[1:] {
				lists = append(lists, toList(x))
			}
			out := []interface{}{[]interface{}{}}
			for _, l := range lists {
				var next []interface{}
				for _, prefix := range out {
					base, _ := prefix.([]interface{})
					for _, item := range l {
						row := append(append([]interface{}{}, base...), item)
						next = append(next, row)
					}
				}
				out = next
			}
			return out, nil
		}),
		fn("zip_longest", func(p ...interface{}) (interface{}, error) {
			pos, kw := splitKwargs(p)
			lists := make([][]interface{}, 0, len(pos))
			longest := 0
			for _, x := range pos {
				l := toList(x)
				lists = append(lists, l)
				if len(l) > longest {
					longest = len(l)
				}
			}
			fill := kw["fillvalue"]
			out := make([]interface{}, 0, longest)
			for i := 0; i < longest; i++ {
				row := make([]interface{}, len(lists))
				for j, l := range lists {
					if i < len(l) {
						row[j] = l[i]
					} else {
						row[j] = fill
					}
				}
				out = append(out, row)
			}
			return out, nil
		}),
		fn("commonpath", func(p ...interface{}) (interface{}, error) {
			paths := toList(p[0])
			if len(paths) == 0 {
				return "", nil
			}
			parts := strings.Split(filepath.Clean(str(paths[0])), "/")
			for _, x := range paths[1:] {
				other := strings.Split(filepath.Clean(str(x)), "/")
				n := 0
				for n < len(parts) && n < len(other) && parts[n] == other[n] {
					n++
				}
				parts = parts[:n]
			}
			out := strings.Join(parts, "/")
			if out == "" && strings.HasPrefix(str(paths[0]), "/") {
				out = "/"
			}
			return out, nil
		}),
		fn("comment", func(p ...interface{}) (interface{}, error) {
			pos, kw := splitKwargs(p)
			text := str(pos[0])
			style := "plain"
			if len(pos) > 1 {
				style = str(pos[1])
			}
			decoration, _ := kw["decoration"].(string)
			prefix, _ := kw["prefix"].(string)
			postfix, _ := kw["postfix"].(string)
			var begin, line, end string
			switch style {
			case "plain":
				line = "# "
			case "erlang":
				line = "% "
			case "c":
				begin, line, end = "/*", " * ", " */"
			case "cblock":
				begin, line, end = "/*", " * ", " */"
			case "xml":
				begin, line, end = "<!--", " - ", "-->"
			default:
				line = "# "
			}
			if decoration != "" {
				line = decoration
			}
			var b strings.Builder
			if prefix != "" {
				b.WriteString(prefix + "\n")
			} else if begin != "" {
				b.WriteString(begin + "\n")
			} else {
				b.WriteString(strings.TrimRight(line, " ") + "\n")
			}
			for _, l := range strings.Split(text, "\n") {
				b.WriteString(strings.TrimRight(line+l, " ") + "\n")
			}
			if postfix != "" {
				b.WriteString(postfix + "\n")
			} else if end != "" {
				b.WriteString(end + "\n")
			} else {
				b.WriteString(strings.TrimRight(line, " ") + "\n")
			}
			return b.String(), nil
		}),
		fn("expanduser", func(p ...interface{}) (interface{}, error) {
			s := str(p[0])
			if strings.HasPrefix(s, "~") {
				if home, err := os.UserHomeDir(); err == nil {
					s = home + strings.TrimPrefix(s, "~")
				}
			}
			return s, nil
		}),
		fn("expandvars", func(p ...interface{}) (interface{}, error) { return os.ExpandEnv(str(p[0])), nil }),
		fn("fileglob", func(p ...interface{}) (interface{}, error) {
			matches, err := filepath.Glob(str(p[0]))
			out := make([]interface{}, 0, len(matches))
			for _, m := range matches {
				if info, err := os.Stat(m); err == nil && !info.IsDir() {
					out = append(out, m)
				}
			}
			return out, err
		}),
		fn("from_yaml_all", func(p ...interface{}) (interface{}, error) {
			dec := yaml.NewDecoder(strings.NewReader(str(p[0])))
			out := []interface{}{}
			for {
				var doc interface{}
				if err := dec.Decode(&doc); err != nil {
					if err.Error() == "EOF" {
						break
					}
					return nil, err
				}
				out = append(out, doc)
			}
			return out, nil
		}),
		fn("human_readable", func(p ...interface{}) (interface{}, error) {
			pos, kw := splitKwargs(p)
			n, ok := toFloat(pos[0])
			if !ok {
				return nil, fmt.Errorf("human_readable: not a number: %v", pos[0])
			}
			isBits := Truthy(kw["isbits"])
			unit := ""
			if u, ok := kw["unit"].(string); ok {
				unit = strings.ToUpper(u)
			}
			suffix := "Bytes"
			base := 1024.0
			if isBits {
				suffix = "bits"
				base = 1000
			}
			units := []string{"", "K", "M", "G", "T", "P", "E"}
			i := 0
			if unit != "" && unit != "AUTO" {
				for j, u := range units {
					if u == unit {
						i = j
					}
				}
				n /= math.Pow(base, float64(i))
			} else {
				for n >= base && i < len(units)-1 {
					n /= base
					i++
				}
			}
			if i == 0 {
				return fmt.Sprintf("%.0f %s", n, suffix), nil
			}
			return fmt.Sprintf("%.2f %s%s", n, units[i], map[bool]string{true: "b", false: "B"}[isBits]), nil
		}),
		fn("human_to_bytes", func(p ...interface{}) (interface{}, error) {
			s := strings.TrimSpace(strings.ToUpper(str(p[0])))
			num := strings.TrimRight(s, "KMGTPEBI ")
			unit := strings.TrimSpace(s[len(num):])
			n, err := strconv.ParseFloat(strings.TrimSpace(num), 64)
			if err != nil {
				return nil, fmt.Errorf("human_to_bytes: %q", p[0])
			}
			mult := map[string]float64{"": 1, "B": 1, "K": 1 << 10, "KB": 1 << 10, "KIB": 1 << 10, "M": 1 << 20, "MB": 1 << 20, "MIB": 1 << 20,
				"G": 1 << 30, "GB": 1 << 30, "GIB": 1 << 30, "T": 1 << 40, "TB": 1 << 40, "TIB": 1 << 40, "P": 1 << 50, "PB": 1 << 50}
			m, ok := mult[unit]
			if !ok {
				return nil, fmt.Errorf("human_to_bytes: unit %q", unit)
			}
			return int(n * m), nil
		}),
		fn("log", func(p ...interface{}) (interface{}, error) {
			n, _ := toFloat(p[0])
			if len(p) > 1 {
				b, _ := toFloat(p[1])
				return math.Log(n) / math.Log(b), nil
			}
			return math.Log(n), nil
		}),
		fn("pow", func(p ...interface{}) (interface{}, error) {
			n, _ := toFloat(p[0])
			e, _ := toFloat(p[1])
			return math.Pow(n, e), nil
		}),
		fn("root", func(p ...interface{}) (interface{}, error) {
			n, _ := toFloat(p[0])
			r := 2.0
			if len(p) > 1 {
				r, _ = toFloat(p[1])
			}
			return math.Pow(n, 1/r), nil
		}),
		fn("normpath", func(p ...interface{}) (interface{}, error) { return filepath.Clean(str(p[0])), nil }),
		fn("path_join", func(p ...interface{}) (interface{}, error) {
			parts := toList(p[0])
			strs := make([]string, len(parts))
			for i, x := range parts {
				strs[i] = str(x)
			}
			return filepath.Join(strs...), nil
		}),
		fn("realpath", func(p ...interface{}) (interface{}, error) {
			r, err := filepath.EvalSymlinks(str(p[0]))
			if err != nil {
				return filepath.Clean(str(p[0])), nil
			}
			return filepath.Abs(r)
		}),
		fn("relpath", func(p ...interface{}) (interface{}, error) {
			base := "."
			if len(p) > 1 {
				base = str(p[1])
			}
			return filepath.Rel(base, str(p[0]))
		}),
		fn("rekey_on_member", func(p ...interface{}) (interface{}, error) {
			pos, kw := splitKwargs(p)
			if len(pos) < 2 {
				return nil, fmt.Errorf("rekey_on_member needs a key")
			}
			key := str(pos[1])
			dup := "error"
			if len(pos) > 2 {
				dup = str(pos[2])
			} else if d, ok := kw["duplicates"].(string); ok {
				dup = d
			}
			out := map[string]interface{}{}
			var items []interface{}
			switch v := pos[0].(type) {
			case map[string]interface{}:
				for _, x := range v {
					items = append(items, x)
				}
			default:
				items = toList(v)
			}
			for _, it := range items {
				m, ok := it.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("rekey_on_member: items must be dicts")
				}
				k := str(m[key])
				if _, seen := out[k]; seen {
					switch dup {
					case "overwrite":
					case "skip":
						continue
					default:
						return nil, fmt.Errorf("rekey_on_member: duplicate key %q", k)
					}
				}
				out[k] = m
			}
			return out, nil
		}),
		fn("strftime", func(p ...interface{}) (interface{}, error) {
			pos, kw := splitKwargs(p)
			t := time.Now()
			if len(pos) > 1 {
				if n, ok := toFloat(pos[1]); ok {
					t = time.Unix(int64(n), 0)
				}
			}
			if Truthy(kw["utc"]) {
				t = t.UTC()
			}
			return pyStrftime(t, str(pos[0])), nil
		}),
		fn("to_datetime", func(p ...interface{}) (interface{}, error) {
			layout := "%Y-%m-%d %H:%M:%S"
			if len(p) > 1 {
				layout = str(p[1])
			}
			t, err := time.Parse(pyToGoLayout(layout), str(p[0]))
			if err != nil {
				return nil, fmt.Errorf("to_datetime: %w", err)
			}
			return t, nil
		}),
		fn("to_uuid", func(p ...interface{}) (interface{}, error) {
			pos, kw := splitKwargs(p)
			ns := uuid.MustParse("361E6D51-FAEC-444A-9079-341391DA2A9F") // Ansible's namespace
			if s, ok := kw["namespace"].(string); ok {
				if u, err := uuid.Parse(s); err == nil {
					ns = u
				}
			}
			return uuid.NewSHA1(ns, []byte(str(pos[0]))).String(), nil
		}),
		fn("type_debug", func(p ...interface{}) (interface{}, error) {
			switch p[0].(type) {
			case nil:
				return "NoneType", nil
			case bool:
				return "bool", nil
			case int, int64, int32:
				return "int", nil
			case float64, float32:
				return "float", nil
			case string:
				return "str", nil
			case []interface{}, []string:
				return "list", nil
			case map[string]interface{}:
				return "dict", nil
			}
			return fmt.Sprintf("%T", p[0]), nil
		}),
		fn("unvault", func(p ...interface{}) (interface{}, error) { return str(p[0]), nil }),
		fn("urldecode", func(p ...interface{}) (interface{}, error) { return url.QueryUnescape(str(p[0])) }),
		fn("urlsplit", func(p ...interface{}) (interface{}, error) {
			u, err := url.Parse(str(p[0]))
			if err != nil {
				return nil, err
			}
			port := interface{}(nil)
			if pt := u.Port(); pt != "" {
				n, _ := strconv.Atoi(pt)
				port = n
			}
			parts := map[string]interface{}{"scheme": u.Scheme, "netloc": u.Host, "hostname": u.Hostname(), "port": port,
				"path": u.Path, "query": u.RawQuery, "fragment": u.Fragment, "username": u.User.Username()}
			if pw, ok := u.User.Password(); ok {
				parts["password"] = pw
			} else {
				parts["password"] = nil
			}
			if len(p) > 1 {
				v, ok := parts[str(p[1])]
				if !ok {
					return nil, fmt.Errorf("urlsplit: unknown component %q", p[1])
				}
				return v, nil
			}
			return parts, nil
		}),
		fn("win_basename", func(p ...interface{}) (interface{}, error) {
			s := str(p[0])
			if i := strings.LastIndexAny(s, `\/`); i >= 0 {
				return s[i+1:], nil
			}
			return s, nil
		}),
		fn("win_dirname", func(p ...interface{}) (interface{}, error) {
			s := str(p[0])
			if i := strings.LastIndexAny(s, `\/`); i >= 0 {
				return s[:i], nil
			}
			return "", nil
		}),
		fn("win_splitdrive", func(p ...interface{}) (interface{}, error) {
			s := str(p[0])
			if len(s) >= 2 && s[1] == ':' {
				return []interface{}{s[:2], s[2:]}, nil
			}
			return []interface{}{"", s}, nil
		}),
		fn("subelements", func(p ...interface{}) (interface{}, error) {
			if len(p) < 2 {
				return nil, fmt.Errorf("subelements needs a key")
			}
			out := []interface{}{}
			for _, item := range toList(p[0]) {
				m, _ := item.(map[string]interface{})
				subs := toList(attribute(m, str(p[1])))
				for _, s := range subs {
					out = append(out, []interface{}{item, s})
				}
			}
			return out, nil
		}),
	}
}

// pyStrftime formats a time with Python's strftime directives
func pyStrftime(t time.Time, layout string) string {
	return t.Format(pyToGoLayout(layout))
}

func pyToGoLayout(layout string) string {
	r := strings.NewReplacer("%Y", "2006", "%m", "01", "%d", "02", "%H", "15", "%M", "04", "%S", "05", "%y", "06",
		"%b", "Jan", "%B", "January", "%a", "Mon", "%A", "Monday", "%p", "PM", "%I", "03", "%j", "002", "%Z", "MST",
		"%z", "-0700", "%f", "000000", "%%", "%")
	return r.Replace(layout)
}

func init() {
	sort.Strings(extraFilterNames)
}
