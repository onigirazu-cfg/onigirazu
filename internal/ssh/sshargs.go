package ssh

import (
	"strconv"
	"strings"
	"time"
)

// sshOptions are the options of ansible_ssh_common_args / ansible_ssh_extra_args
// that onigirazu uses; others are ignored
type sshOptions struct {
	ConnectTimeout time.Duration
	ProxyJump      string // [user@]host[:port][,...], as ssh -J
	ProxyCommand   string // %h, %p and %r are replaced
}

// parseSSHArgs reads -o Key=Value, -oKey=Value, -o "Key Value" and -J host
func parseSSHArgs(args string) sshOptions {
	var o sshOptions
	words := shellWords(args)
	for i := 0; i < len(words); i++ {
		w := words[i]
		switch {
		case w == "-J" && i+1 < len(words):
			i++
			o.ProxyJump = words[i]
			continue
		case strings.HasPrefix(w, "-J") && len(w) > 2:
			o.ProxyJump = w[2:]
			continue
		case w == "-o" && i+1 < len(words):
			i++
			w = words[i]
		case strings.HasPrefix(w, "-o"):
			w = w[2:]
		default:
			continue
		}
		key, value, ok := strings.Cut(w, "=")
		if !ok {
			key, value, _ = strings.Cut(w, " ")
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "connecttimeout":
			if n, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && n > 0 {
				o.ConnectTimeout = time.Duration(n) * time.Second
			}
		case "proxyjump":
			o.ProxyJump = strings.TrimSpace(value)
		case "proxycommand":
			o.ProxyCommand = strings.TrimSpace(value)
		}
	}
	if strings.EqualFold(o.ProxyJump, "none") {
		o.ProxyJump = ""
	}
	if strings.EqualFold(o.ProxyCommand, "none") {
		o.ProxyCommand = ""
	}
	return o
}

// shellWords splits like a POSIX shell: quotes group, backslash escapes
func shellWords(s string) []string {
	var words []string
	var cur strings.Builder
	inWord := false
	var quote rune
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			inWord = true
		case r == ' ' || r == '\t' || r == '\n':
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words
}
