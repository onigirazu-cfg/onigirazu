package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// import_playbook: a playbook item "- import_playbook: other.yml" is
// replaced by the plays of that file, relative to the playbook that imports
// it. vars on the import go to the imported plays (a play's own vars win),
// tags are added to them. Every play keeps the directory of the file it
// came from: its roles, includes, vars_files and task sources start there.
// The plays are moved as YAML nodes, so their values keep their spelling
// (mode: 0644).

const maxImportDepth = 16

// expandImportPlaybooks returns the playbook with its imports replaced by
// their plays, and the directory each play came from (nil: no imports)
func expandImportPlaybooks(content []byte, dir string, depth int) ([]byte, []string, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.SequenceNode {
		return content, nil, nil // the plays: form, or an error the parser reports
	}
	plays, dirs, found, err := importItems(doc.Content[0], dir, depth)
	if err != nil || !found {
		return content, nil, err
	}
	out, err := yaml.Marshal(&yaml.Node{Kind: yaml.SequenceNode, Content: plays})
	if err != nil {
		return nil, nil, err
	}
	return out, dirs, nil
}

func importItems(seq *yaml.Node, dir string, depth int) ([]*yaml.Node, []string, bool, error) {
	if depth > maxImportDepth {
		return nil, nil, false, fmt.Errorf("import_playbook nested deeper than %d", maxImportDepth)
	}
	var plays []*yaml.Node
	var dirs []string
	found := false
	for _, item := range seq.Content {
		target, isImport := importTarget(item)
		if !isImport {
			plays = append(plays, item)
			dirs = append(dirs, dir)
			continue
		}
		found = true
		if mapValue(item, "when") != nil {
			return nil, nil, false, fmt.Errorf("import_playbook %s: when is not supported", target)
		}
		if strings.Contains(target, "{{") {
			return nil, nil, false, fmt.Errorf("import_playbook %s: the path cannot be a template", target)
		}
		path := target
		if !filepath.IsAbs(path) {
			path = filepath.Join(dir, path)
		}
		data, err := os.ReadFile(path) // #nosec G304 -- a playbook the playbook imports
		if err != nil {
			return nil, nil, false, fmt.Errorf("import_playbook: %w", err)
		}
		var sub yaml.Node
		if err := yaml.Unmarshal(data, &sub); err != nil {
			return nil, nil, false, fmt.Errorf("import_playbook %s: %w", target, err)
		}
		if len(sub.Content) == 0 || sub.Content[0].Kind != yaml.SequenceNode {
			return nil, nil, false, fmt.Errorf("import_playbook %s: not a list of plays", target)
		}
		subPlays, subDirs, _, err := importItems(sub.Content[0], filepath.Dir(path), depth+1)
		if err != nil {
			return nil, nil, false, err
		}
		vars, tags := mapValue(item, "vars"), mapValue(item, "tags")
		for _, play := range subPlays {
			applyImportOptions(play, vars, tags)
		}
		plays = append(plays, subPlays...)
		dirs = append(dirs, subDirs...)
	}
	return plays, dirs, found, nil
}

func importTarget(item *yaml.Node) (string, bool) {
	for _, key := range []string{"import_playbook", "ansible.builtin.import_playbook"} {
		if v := mapValue(item, key); v != nil {
			return v.Value, true
		}
	}
	return "", false
}

// mapValue is the value node of key in a mapping node
func mapValue(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// applyImportOptions gives an imported play the vars and tags of its import
func applyImportOptions(play, vars, tags *yaml.Node) {
	if play.Kind != yaml.MappingNode {
		return
	}
	if vars != nil && vars.Kind == yaml.MappingNode {
		own := mapValue(play, "vars")
		if own == nil {
			own = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			play.Content = append(play.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "vars"}, own)
		}
		for i := 0; i+1 < len(vars.Content); i += 2 {
			if mapValue(own, vars.Content[i].Value) == nil {
				own.Content = append(own.Content, vars.Content[i], vars.Content[i+1])
			}
		}
	}
	if tags != nil {
		own := mapValue(play, "tags")
		if own == nil {
			play.Content = append(play.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "tags"}, tags)
			return
		}
		list := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, t := range []*yaml.Node{own, tags} {
			if t.Kind == yaml.SequenceNode {
				list.Content = append(list.Content, t.Content...)
			} else {
				list.Content = append(list.Content, t)
			}
		}
		*own = *list
	}
}

// playDir is where a play's relative paths start
func playDir(play *types.Play, fallback string) string {
	if play.BaseDir != "" {
		return play.BaseDir
	}
	return fallback
}
