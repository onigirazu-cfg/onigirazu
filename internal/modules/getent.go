package modules

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

var getentDatabase = regexp.MustCompile(`^[a-z_]+$`)

const getentRCMarker = "ONIGIRAZU-GETENT-RC "

// GetentModule reads a getent database into the getent_<database> fact
// (Ansible's getent): the first field is the key, the others its value
type GetentModule struct {
	*BaseModule
}

// NewGetentModule creates the getent module
func NewGetentModule() *GetentModule {
	return &GetentModule{BaseModule: NewBaseModule("getent")}
}

func (m *GetentModule) GetDescription() string { return "Read a getent database (passwd, group, hosts, ...)" }

func (m *GetentModule) Validate(args map[string]interface{}) error {
	if !getentDatabase.MatchString(getStringArg(args, "database", "")) {
		return fmt.Errorf("getent requires 'database' (passwd, group, hosts, ...)")
	}
	return nil
}

func (m *GetentModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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
	database := getStringArg(args, "database", "")
	key := getStringArg(args, "key", "")
	cmd := "getent " + shellQuote(database)
	if key != "" {
		cmd += " " + shellQuote(key)
	}
	out, err := runShellOnHost(ctx, host, args, cmd+" || echo "+getentRCMarker+"$?")
	if err != nil {
		return fail(fmt.Sprintf("getent failed: %v", err))
	}
	if i := strings.Index(out, getentRCMarker); i >= 0 {
		rc := strings.TrimSpace(out[i+len(getentRCMarker):])
		out = out[:i]
		switch {
		case rc == "2" && key != "" && !getBoolArg(args, "fail_key", true):
			// a missing key is fine: the fact has no entry for it
		case rc == "2" && key != "":
			return fail(fmt.Sprintf("One or more supplied key could not be found in the database: %s", key))
		case rc == "2":
		default:
			return fail(fmt.Sprintf("getent %s failed with exit code %s", database, rc))
		}
	}
	entries := parseGetent(out, database, getStringArg(args, "split", ""))
	facts := map[string]interface{}{"getent_" + database: entries}
	result.Output["ansible_facts"] = facts
	result.Output["onigirazu_facts"] = facts
	result.Duration = time.Since(start)
	return result, nil
}

// parseGetent turns getent lines into key -> other fields: split by ":"
// for the colon databases, by whitespace for hosts, services, ...
func parseGetent(out, database, split string) map[string]interface{} {
	if split == "" {
		switch database {
		case "passwd", "shadow", "group", "gshadow":
			split = ":"
		}
	}
	entries := map[string]interface{}{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		var fields []string
		if split == "" {
			fields = strings.Fields(line)
		} else {
			fields = strings.Split(line, split)
		}
		value := make([]interface{}, 0, len(fields)-1)
		for _, f := range fields[1:] {
			value = append(value, f)
		}
		entries[fields[0]] = value
	}
	return entries
}
