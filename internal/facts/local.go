package facts

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
)

// DefaultFactPath is where Ansible reads local facts from
const DefaultFactPath = "/etc/ansible/facts.d"

const localFactMarker = "==ONIGIRAZU-LOCAL-FACT "

// localFactsScript prints every *.fact file of dir behind a marker line; an
// executable one is run and its output printed instead
func localFactsScript(dir string) string {
	q := "'" + strings.ReplaceAll(dir, "'", `'\''`) + "'"
	return fmt.Sprintf(`d=%s; [ -d "$d" ] || exit 0
for f in "$d"/*.fact; do
  [ -f "$f" ] || continue
  printf '\n%%s%%s\n' '%s' "$(basename "$f" .fact)"
  if [ -x "$f" ]; then "$f" 2>/dev/null; else cat "$f" 2>/dev/null; fi
done`, q, localFactMarker)
}

// gatherLocalFacts reads ansible_local: each file's JSON, else its INI
// sections, else its text
func (g *Gatherer) gatherLocalFacts(client commandRunner, facts *cache.SystemFacts, dir string) {
	facts.Local = map[string]interface{}{}
	if facts.Kernel != "Linux" && facts.Kernel != "Darwin" {
		return
	}
	out, err := client.ExecuteCommand(localFactsScript(dir))
	if err != nil {
		return
	}
	facts.Local = parseLocalFacts(out)
}

func parseLocalFacts(out string) map[string]interface{} {
	result := map[string]interface{}{}
	parts := strings.Split(out, "\n"+localFactMarker)
	for _, part := range parts[1:] {
		name, body, _ := strings.Cut(part, "\n")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		result[name] = parseLocalFact(body)
	}
	return result
}

func parseLocalFact(body string) interface{} {
	var value interface{}
	if err := json.Unmarshal([]byte(body), &value); err == nil {
		return value
	}
	if ini, ok := parseINIFact(body); ok {
		return ini
	}
	return strings.TrimSpace(body)
}

// parseINIFact reads [section] and key=value lines, as Ansible does for a
// .fact file that is not JSON
func parseINIFact(body string) (map[string]interface{}, bool) {
	result := map[string]interface{}{}
	var section map[string]interface{}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = map[string]interface{}{}
			result[strings.TrimSpace(line[1:len(line)-1])] = section
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			key, value, ok = strings.Cut(line, ":")
		}
		if !ok || section == nil {
			return nil, false
		}
		section[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return result, len(result) > 0
}
