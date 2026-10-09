package modules

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// IniFileModule sets or removes one option of an INI file (Ansible's
// ini_file): path, section (none: before the first section), option, value,
// state present/absent, no_extra_spaces, create, allow_no_value, backup, mode
type IniFileModule struct {
	*BaseModule
}

// NewIniFileModule creates the ini_file module
func NewIniFileModule() *IniFileModule {
	return &IniFileModule{BaseModule: NewBaseModule("ini_file")}
}

func (m *IniFileModule) GetDescription() string { return "Manage one option of an INI file" }

func (m *IniFileModule) Validate(args map[string]interface{}) error {
	if filePath(args) == "" {
		return fmt.Errorf("ini_file requires 'path'")
	}
	state := getStringArg(args, "state", "present")
	if state != "present" && state != "absent" {
		return fmt.Errorf("ini_file: state must be present or absent")
	}
	if state == "present" && getStringArg(args, "option", "") != "" && args["value"] == nil && !getBoolArg(args, "allow_no_value", false) {
		return fmt.Errorf("ini_file: 'value' is required with an option (or allow_no_value)")
	}
	return nil
}

func (m *IniFileModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name,
		Timestamp: start, Success: true, Output: map[string]interface{}{}}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error, result.Duration = false, msg, time.Since(start)
		return result, nil
	}
	if err := m.Validate(args); err != nil {
		return fail(err.Error())
	}
	path := filePath(args)
	data, exists, err := readHostFile(ctx, host, args, path)
	if err != nil {
		return fail(err.Error())
	}
	if !exists && !getBoolArg(args, "create", true) {
		return fail(fmt.Sprintf("destination %s does not exist", path))
	}
	var value *string
	if v, ok := args["value"]; ok && v != nil {
		s := fmt.Sprint(v)
		value = &s
	}
	after := editIni(string(data), getStringArg(args, "section", ""), getStringArg(args, "option", ""), value,
		getStringArg(args, "state", "present") == "present", getBoolArg(args, "no_extra_spaces", false))
	result.Output["path"] = path
	if after == string(data) && exists {
		result.Output["msg"] = "OK"
		result.Duration = time.Since(start)
		return result, nil
	}
	result.Changed = true
	result.Output["msg"] = "option changed"
	addDiff(args, &result, path, diffText(data, exists), diffText([]byte(after), true))
	if inCheckMode(args) {
		result.Duration = time.Since(start)
		return result, nil
	}
	if exists && getBoolArg(args, "backup", false) {
		backup := fmt.Sprintf("%s.%s~", path, time.Now().Format("20060102150405"))
		if err := writeHostFile(ctx, host, args, backup, data, 0); err != nil {
			return fail(err.Error())
		}
		result.Output["backup_file"] = backup
	}
	var mode os.FileMode
	if s := getStringArg(args, "mode", ""); s != "" {
		if n, err := strconv.ParseUint(s, 8, 32); err == nil {
			mode = os.FileMode(n)
		}
	}
	if err := validateBeforeWrite(ctx, host, args, path, []byte(after)); err != nil {
		return fail(err.Error())
	}
	if err := writeHostFile(ctx, host, args, path, []byte(after), mode); err != nil {
		return fail(err.Error())
	}
	result.Duration = time.Since(start)
	return result, nil
}

// editIni sets (present) or removes the option in the section and returns
// the new text; an empty option with present only makes sure the section
// exists, with absent removes the section
func editIni(text, section, option string, value *string, present, noSpaces bool) string {
	lines := strings.SplitAfter(text, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > 0 && !strings.HasSuffix(lines[len(lines)-1], "\n") {
		lines[len(lines)-1] += "\n"
	}
	assign := " = "
	if noSpaces {
		assign = "="
	}
	newLine := option + "\n"
	if value != nil {
		newLine = option + assign + *value + "\n"
	}

	// the lines of the section: [start, end); start 0 and no header for none
	start, end, found := 0, len(lines), section == ""
	if section == "" {
		for i, l := range lines {
			if isSectionHeader(l) {
				end = i
				break
			}
		}
	} else {
		for i, l := range lines {
			if isSectionHeader(l) && sectionName(l) == section {
				start, found = i+1, true
				end = len(lines)
				for j := i + 1; j < len(lines); j++ {
					if isSectionHeader(lines[j]) {
						end = j
						break
					}
				}
				break
			}
		}
	}

	if option == "" {
		switch {
		case present && !found:
			return strings.Join(lines, "") + "[" + section + "]\n"
		case !present && found && section != "":
			return strings.Join(append(append([]string{}, lines[:start-1]...), lines[end:]...), "")
		}
		return strings.Join(lines, "")
	}

	if !found {
		if !present {
			return strings.Join(lines, "")
		}
		out := strings.Join(lines, "")
		if out != "" && !strings.HasSuffix(out, "\n\n") {
			out += "\n"
		}
		if strings.TrimSpace(strings.Join(lines, "")) == "" {
			out = ""
		}
		return out + "[" + section + "]\n" + newLine
	}

	var body []string
	placed := false
	for _, l := range lines[start:end] {
		if optionName(l) == option {
			if present && !placed {
				body = append(body, newLine)
				placed = true
			}
			continue // exclusive: other occurrences go
		}
		body = append(body, l)
	}
	if present && !placed {
		// after the last non-blank line of the section
		at := len(body)
		for at > 0 && strings.TrimSpace(body[at-1]) == "" {
			at--
		}
		body = append(body[:at], append([]string{newLine}, body[at:]...)...)
	}
	out := append(append(append([]string{}, lines[:start]...), body...), lines[end:]...)
	return strings.Join(out, "")
}

func isSectionHeader(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]")
}

func sectionName(line string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(line), "["), "]"))
}

// optionName is the option a line sets ("" for comments and blanks)
func optionName(line string) string {
	t := strings.TrimSpace(line)
	if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, ";") {
		return ""
	}
	if i := strings.IndexAny(t, "=:"); i >= 0 {
		return strings.TrimSpace(t[:i])
	}
	return t
}
