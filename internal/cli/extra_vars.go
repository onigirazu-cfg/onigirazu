package cli

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// parseExtraVars reads -e values the way Ansible does: "key=value" pairs
// separated by spaces (values are strings), inline JSON or YAML ("{...}"),
// or "@file" with YAML/JSON. Later values override earlier ones.
func parseExtraVars(values []string) (map[string]interface{}, error) {
	vars := make(map[string]interface{})
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		var parsed map[string]interface{}
		switch {
		case value == "":
			continue
		case strings.HasPrefix(value, "@"):
			data, err := os.ReadFile(strings.TrimPrefix(value, "@")) // #nosec G304 -- the user names the file
			if err != nil {
				return nil, fmt.Errorf("-e %s: %w", value, err)
			}
			if err := yaml.Unmarshal(data, &parsed); err != nil {
				return nil, fmt.Errorf("-e %s: %w", value, err)
			}
		case strings.HasPrefix(value, "{"):
			if err := yaml.Unmarshal([]byte(value), &parsed); err != nil {
				return nil, fmt.Errorf("-e %s: %w", value, err)
			}
		default:
			parsed = make(map[string]interface{})
			for _, pair := range strings.Fields(value) {
				k, v, ok := strings.Cut(pair, "=")
				if !ok || k == "" {
					return nil, fmt.Errorf("-e %q: expected key=value", pair)
				}
				parsed[k] = v
			}
		}
		for k, v := range parsed {
			vars[k] = v
		}
	}
	return vars, nil
}
