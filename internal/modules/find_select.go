package modules

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// findSelection is what Ansible's find selects beyond names and type:
// age, size, hidden entries, excludes, regex patterns and the depth
type findSelection struct {
	age, size         float64 // seconds / bytes; negative = at most, positive = at least
	hasAge, hasSize   bool
	hidden            bool
	useRegex          bool
	patterns, exclude []*regexp.Regexp
	excludeGlobs      []string
	depth             int
}

func newFindSelection(args map[string]interface{}, patterns []string) (*findSelection, error) {
	s := &findSelection{hidden: getBoolArg(args, "hidden", false), useRegex: getBoolArg(args, "use_regex", false), depth: getIntArg(args, "depth", 0)}
	// an option that is not honored fails instead of widening the match
	if v := getStringArg(args, "contains", ""); v != "" {
		return nil, fmt.Errorf("find: 'contains' is not supported")
	}
	if v := getStringArg(args, "age_stamp", "mtime"); v != "mtime" {
		return nil, fmt.Errorf("find: age_stamp %q is not supported (only mtime)", v)
	}
	if v := getStringArg(args, "age", ""); v != "" {
		age, err := parseFindAmount(v, map[string]float64{"s": 1, "m": 60, "h": 3600, "d": 86400, "w": 604800}, "s")
		if err != nil {
			return nil, fmt.Errorf("find: invalid age %q", v)
		}
		s.age, s.hasAge = age, true
	}
	if v := getStringArg(args, "size", ""); v != "" {
		size, err := parseFindAmount(strings.ToLower(v), map[string]float64{"b": 1, "k": 1 << 10, "m": 1 << 20, "g": 1 << 30, "t": 1 << 40}, "b")
		if err != nil {
			return nil, fmt.Errorf("find: invalid size %q", v)
		}
		s.size, s.hasSize = size, true
	}
	excludes := listArg(args, "excludes", "exclude")
	if s.useRegex {
		for _, p := range patterns {
			re, err := regexp.Compile("^(?:" + p + ")")
			if err != nil {
				return nil, fmt.Errorf("find: invalid pattern %q: %v", p, err)
			}
			s.patterns = append(s.patterns, re)
		}
		for _, p := range excludes {
			re, err := regexp.Compile("^(?:" + p + ")")
			if err != nil {
				return nil, fmt.Errorf("find: invalid exclude %q: %v", p, err)
			}
			s.exclude = append(s.exclude, re)
		}
	} else {
		s.excludeGlobs = excludes
	}
	return s, nil
}

// parseFindAmount reads "30d", "-2h", "10m" with the given units
func parseFindAmount(v string, units map[string]float64, def string) (float64, error) {
	unit := def
	if n := len(v); n > 0 {
		if _, ok := units[v[n-1:]]; ok {
			unit, v = v[n-1:], v[:n-1]
		}
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, err
	}
	return n * units[unit], nil
}

// keep applies the selection to a record of parseFindRecord
func (s *findSelection) keep(f map[string]interface{}, now float64) bool {
	name, _ := f["name"].(string)
	if !s.hidden && strings.HasPrefix(name, ".") {
		return false
	}
	if s.useRegex {
		matched := false
		for _, re := range s.patterns {
			matched = matched || re.MatchString(name)
		}
		if !matched {
			return false
		}
		for _, re := range s.exclude {
			if re.MatchString(name) {
				return false
			}
		}
	}
	for _, g := range s.excludeGlobs {
		if ok, _ := filepath.Match(g, name); ok {
			return false
		}
	}
	if s.hasAge {
		mtime, ok := f["mtime"].(float64)
		if !ok {
			return false
		}
		elapsed := now - mtime
		if (s.age >= 0 && elapsed < s.age) || (s.age < 0 && elapsed > -s.age) {
			return false
		}
	}
	if s.hasSize {
		size, ok := f["size"].(int64)
		if !ok {
			return false
		}
		if (s.size >= 0 && float64(size) < s.size) || (s.size < 0 && float64(size) > -s.size) {
			return false
		}
	}
	return true
}
