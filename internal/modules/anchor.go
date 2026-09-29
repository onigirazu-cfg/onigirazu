package modules

import (
	"fmt"
	"regexp"
)

// anchorIndex returns the index in lines at which new content goes for the
// insertafter/insertbefore arguments, the way Ansible places it: both are
// regular expressions, the last matching line wins (the first one with
// firstmatch), EOF and BOF are keywords, and no match means the end of the file.
func anchorIndex(lines []string, insertafter, insertbefore string, firstmatch bool) (int, error) {
	switch {
	case insertbefore == "BOF":
		return 0, nil
	case insertbefore == "" && (insertafter == "" || insertafter == "EOF"):
		return len(lines), nil
	}
	expr, after := insertbefore, false
	if expr == "" {
		expr, after = insertafter, true
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return 0, fmt.Errorf("invalid insertafter/insertbefore pattern %q: %v", expr, err)
	}
	found := -1
	for i, l := range lines {
		if re.MatchString(l) {
			found = i
			if firstmatch {
				break
			}
		}
	}
	switch {
	case found < 0:
		return len(lines), nil
	case after:
		return found + 1, nil
	default:
		return found, nil
	}
}
