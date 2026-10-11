package logger

import (
	"sort"
	"strings"
	"sync"
)

// Secrets never belong in a log line: values read from a secret backend
// and the passwords of an inventory are registered here, and every logger
// replaces them before writing. Short values are not registered: masking
// "1" or "ab" would damage unrelated text.

const redacted = "***"

var (
	redactMu sync.RWMutex
	secrets  []string
	known    = map[string]bool{}
)

// Redact registers a value to hide from every log line
func Redact(value string) {
	if len(value) < 4 || strings.TrimSpace(value) == "" {
		return
	}
	redactMu.Lock()
	defer redactMu.Unlock()
	if known[value] {
		return
	}
	known[value] = true
	secrets = append(secrets, value)
	// longest first: a secret that contains another is replaced whole
	sort.Slice(secrets, func(i, j int) bool { return len(secrets[i]) > len(secrets[j]) })
}

// Scrub returns the text with every registered secret replaced
func Scrub(text string) string {
	redactMu.RLock()
	defer redactMu.RUnlock()
	for _, s := range secrets {
		if strings.Contains(text, s) {
			text = strings.ReplaceAll(text, s, redacted)
		}
	}
	return text
}

// resetRedaction forgets every registered secret (tests)
func resetRedaction() {
	redactMu.Lock()
	defer redactMu.Unlock()
	secrets, known = nil, map[string]bool{}
}
