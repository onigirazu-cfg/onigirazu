package expression

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ansibleTest covers the Ansible tests beyond Jinja's: task results
// (success, failed, changed, skipped), version comparison, types, sets and
// paths on the control machine. handled is false for other tests.
func ansibleTest(test string, v interface{}, args []interface{}) (ok, handled bool, err error) {
	arg := func(i int) interface{} {
		if i < len(args) {
			return args[i]
		}
		return nil
	}
	result, _ := v.(map[string]interface{})
	switch test {
	case "success", "succeeded", "successful":
		return result != nil && !Truthy(result["failed"]) && !Truthy(result["unreachable"]), true, nil
	case "reachable":
		return result != nil && !Truthy(result["unreachable"]), true, nil
	case "finished":
		return result != nil && (Truthy(result["finished"]) || result["finished"] == nil && !Truthy(result["started"])), true, nil
	case "started":
		return result != nil && (Truthy(result["started"]) || Truthy(result["finished"])), true, nil
	case "timedout":
		return result != nil && (Truthy(result["timedout"]) || strings.Contains(strings.ToLower(PyStr(result["msg"])), "timed out")), true, nil
	case "all":
		for _, x := range toList(v) {
			if !Truthy(x) {
				return false, true, nil
			}
		}
		return true, true, nil
	case "any":
		for _, x := range toList(v) {
			if Truthy(x) {
				return true, true, nil
			}
		}
		return false, true, nil
	case "nan", "isnan":
		f, ok := toFloat(v)
		return ok && f != f, true, nil
	case "uri", "url", "urn":
		s := fmt.Sprint(v)
		scheme, rest, has := strings.Cut(s, ":")
		if !has || scheme == "" || strings.ContainsAny(scheme, " /") {
			return false, true, nil
		}
		schemes := toList(arg(0))
		if len(schemes) > 0 {
			ok := false
			for _, x := range schemes {
				if strings.EqualFold(fmt.Sprint(x), scheme) {
					ok = true
				}
			}
			if !ok {
				return false, true, nil
			}
		}
		switch test {
		case "url":
			return strings.HasPrefix(rest, "//"), true, nil
		case "urn":
			return strings.EqualFold(scheme, "urn") && rest != "", true, nil
		}
		return rest != "", true, nil
	case "vault_encrypted":
		return strings.HasPrefix(strings.TrimSpace(fmt.Sprint(v)), "$ANSIBLE_VAULT;"), true, nil
	case "vaulted_file":
		data, err := os.ReadFile(fmt.Sprint(v)) // #nosec G304 -- the playbook's own file
		return err == nil && strings.HasPrefix(string(data), "$ANSIBLE_VAULT;"), true, nil
	case "same_file", "is_same_file":
		a, errA := os.Stat(fmt.Sprint(v))
		b, errB := os.Stat(fmt.Sprint(arg(0)))
		return errA == nil && errB == nil && os.SameFile(a, b), true, nil
	case "mount", "is_mount":
		p := filepath.Clean(fmt.Sprint(v))
		info, err := os.Stat(p)
		if err != nil {
			return false, true, nil
		}
		parent, err := os.Stat(filepath.Dir(p))
		if err != nil {
			return false, true, nil
		}
		return p == "/" || !os.SameFile(info, parent) && deviceOf(info) != deviceOf(parent), true, nil
	case "failed", "failure":
		return result != nil && Truthy(result["failed"]), true, nil
	case "changed", "change":
		return result != nil && Truthy(result["changed"]), true, nil
	case "skipped", "skip":
		return result != nil && Truthy(result["skipped"]), true, nil
	case "unreachable":
		return result != nil && Truthy(result["unreachable"]), true, nil
	case "version", "version_compare":
		op := "=="
		if o := arg(1); o != nil {
			op = fmt.Sprint(o)
		}
		c := compareVersions(fmt.Sprint(v), fmt.Sprint(arg(0)))
		switch op {
		case "==", "=", "eq":
			return c == 0, true, nil
		case "!=", "<>", "ne":
			return c != 0, true, nil
		case "<", "lt":
			return c < 0, true, nil
		case "<=", "le":
			return c <= 0, true, nil
		case ">", "gt":
			return c > 0, true, nil
		case ">=", "ge":
			return c >= 0, true, nil
		}
		return false, true, fmt.Errorf("version: unknown operator %q", op)
	case "mapping":
		_, ok := v.(map[string]interface{})
		return ok, true, nil
	case "sequence", "iterable":
		switch v.(type) {
		case []interface{}, []string, string, map[string]interface{}:
			return true, true, nil
		}
		return false, true, nil
	case "boolean":
		_, ok := v.(bool)
		return ok, true, nil
	case "integer":
		switch n := v.(type) {
		case int, int64:
			return true, true, nil
		case float64:
			return n == float64(int64(n)), true, nil
		}
		return false, true, nil
	case "float":
		_, ok := v.(float64)
		return ok, true, nil
	case "lower":
		s := fmt.Sprint(v)
		return s == strings.ToLower(s), true, nil
	case "upper":
		s := fmt.Sprint(v)
		return s == strings.ToUpper(s), true, nil
	case "even", "odd", "divisibleby":
		n, ok := number(v)
		if !ok {
			return false, true, nil
		}
		d := 2.0
		if test == "divisibleby" {
			if d, ok = number(arg(0)); !ok || d == 0 {
				return false, true, nil
			}
		}
		rem := int64(n) % int64(d)
		if test == "odd" {
			return rem != 0, true, nil
		}
		return rem == 0, true, nil
	case "subset", "superset", "issubset", "issuperset":
		a, b := toList(v), toList(arg(0))
		if test == "superset" || test == "issuperset" {
			a, b = b, a
		}
		for _, x := range a {
			if !contains(b, x) {
				return false, true, nil
			}
		}
		return true, true, nil
	case "exists", "file", "directory", "link", "abs", "is_file", "is_dir", "is_link", "is_abs", "link_exists":
		p := fmt.Sprint(v)
		if test == "abs" || test == "is_abs" {
			return filepath.IsAbs(p), true, nil
		}
		info, err := os.Lstat(p)
		if err != nil {
			return false, true, nil
		}
		switch test {
		case "file", "is_file":
			return info.Mode().IsRegular(), true, nil
		case "directory", "is_dir":
			return info.IsDir(), true, nil
		case "link", "is_link", "link_exists":
			return info.Mode()&os.ModeSymlink != 0, true, nil
		}
		return true, true, nil
	}
	return false, false, nil
}

func toList(v interface{}) []interface{} {
	switch l := v.(type) {
	case []interface{}:
		return l
	case []string:
		out := make([]interface{}, len(l))
		for i, s := range l {
			out[i] = s
		}
		return out
	}
	return nil
}

var versionPart = regexp.MustCompile(`\d+|[A-Za-z]+`)

// compareVersions compares like Python's LooseVersion: numbers as numbers,
// words as text, a missing part is smaller
func compareVersions(a, b string) int {
	pa, pb := versionPart.FindAllString(a, -1), versionPart.FindAllString(b, -1)
	for i := 0; i < len(pa) || i < len(pb); i++ {
		if i >= len(pa) {
			return -1
		}
		if i >= len(pb) {
			return 1
		}
		na, ea := strconv.Atoi(pa[i])
		nb, eb := strconv.Atoi(pb[i])
		switch {
		case ea == nil && eb == nil:
			if na != nb {
				if na < nb {
					return -1
				}
				return 1
			}
		case ea == nil:
			return 1 // a number is newer than a word (1.0 > 1.0rc)
		case eb == nil:
			return -1
		default:
			if c := strings.Compare(pa[i], pb[i]); c != 0 {
				return c
			}
		}
	}
	return 0
}
