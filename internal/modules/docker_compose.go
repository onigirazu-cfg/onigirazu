package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type DockerComposeModule struct {
	BaseModule
}

func NewDockerComposeModule() *DockerComposeModule {
	return &DockerComposeModule{
		BaseModule: BaseModule{
			name:        "docker_compose",
			description: "Manage Docker Compose applications",
		},
	}
}

func (m *DockerComposeModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  taskName(args),
		Host:      host.Name,
		Module:    m.GetName(),
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}

	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to create executor: %v", err)
		return result, err
	}
	defer exec.Close()

	composeV2Args(args)
	projectDir, ok := args["project_dir"].(string)
	if !ok || projectDir == "" {
		result.Success = false
		result.Error = "project_dir is required"
		return result, fmt.Errorf("project_dir is required")
	}
	p := newComposeProject(projectDir, args)

	state, _ := args["state"].(string)
	if state == "" {
		state = "present"
	}

	run := func(action, sub string) error {
		before := m.snapshot(exec, p)
		if _, err := exec.Execute(m.buildComposeCmd(p, sub)); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("compose %s failed: %v", action, err)
			return err
		}
		result.Changed = m.snapshot(exec, p) != before
		return nil
	}

	err = nil
	switch state {
	case "present":
		err = run("up", upCommand(args))
		result.Output["action"] = "started"
	case "absent":
		err = run("down", downCommand(args))
		result.Output["action"] = "stopped"
	case "stopped":
		err = run("stop", "stop"+servicesArg(args))
		result.Output["action"] = "stopped"
	case "restarted":
		if _, err = exec.Execute(m.buildComposeCmd(p, "restart"+servicesArg(args))); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("compose restart failed: %v", err)
		}
		result.Changed = true
		result.Output["action"] = "restarted"
	case "pull":
		if _, err = exec.Execute(m.buildComposeCmd(p, "pull"+servicesArg(args))); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("compose pull failed: %v", err)
		}
		result.Changed = true
		result.Output["action"] = "pulled"
	case "build":
		sub := "build"
		if getBoolArg(args, "nocache", false) {
			sub += " --no-cache"
		}
		if b, _ := args["pull"].(bool); b {
			sub += " --pull"
		}
		if _, err = exec.Execute(m.buildComposeCmd(p, sub+servicesArg(args))); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("compose build failed: %v", err)
		}
		result.Changed = true
		result.Output["action"] = "built"
	default:
		err = fmt.Errorf("state must be present, absent, stopped, restarted, pull or build, got %q", state)
		result.Success = false
		result.Error = err.Error()
	}
	if err != nil {
		result.Duration = time.Since(startTime)
		return result, err
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// composeProject is where compose runs and its global flags (-f, -p,
// --profile, --env-file), already quoted
type composeProject struct {
	dir   string
	flags []string
}

func newComposeProject(dir string, args map[string]interface{}) composeProject {
	p := composeProject{dir: dir}
	add := func(flag string, values []string) {
		for _, v := range values {
			p.flags = append(p.flags, flag, shellQuote(v))
		}
	}
	add("-f", listArg(args, "files", "file"))
	if name, ok := args["project_name"].(string); ok && name != "" {
		add("-p", []string{name})
	}
	add("--profile", listArg(args, "profiles"))
	add("--env-file", listArg(args, "env_files"))
	return p
}

// buildComposeCmd prefers the Compose v2 plugin and falls back to the v1
// docker-compose binary. Files are passed only when set, so compose.yaml and
// docker-compose.yml are both found by default.
func (m *DockerComposeModule) buildComposeCmd(p composeProject, sub string) string {
	script := `c() { if docker compose version >/dev/null 2>&1; then docker compose "$@"; else docker-compose "$@"; fi; }; ` +
		"cd " + shellQuote(p.dir) + " && c " + strings.Join(append(append([]string{}, p.flags...), sub), " ")
	return "sh -c " + shellQuote(script)
}

// snapshot lists all project containers and the running ones; comparing it
// before and after tells whether a command changed anything
func (m *DockerComposeModule) snapshot(exec *executor.CommandExecutor, p composeProject) string {
	out, err := exec.Execute(m.buildComposeCmd(p, "ps -a -q; echo --; c "+strings.Join(p.flags, " ")+" ps -q"))
	if err != nil {
		return "error: " + err.Error()
	}
	return out
}

func servicesArg(args map[string]interface{}) string {
	s := ""
	for _, svc := range listArg(args, "services") {
		s += " " + shellQuote(svc)
	}
	return s
}

// upCommand is "up" with the options of docker_compose (booleans) and of
// docker_compose_v2 (build/pull/recreate policies, wait, remove_orphans)
func upCommand(args map[string]interface{}) string {
	parts := []string{"up"}
	if detach, set := args["detach"]; !set || detach == nil || getBoolArg(args, "detach", true) {
		parts = append(parts, "-d")
	}
	switch v := args["build"].(type) {
	case bool:
		if v {
			parts = append(parts, "--build")
		}
	case string:
		switch v {
		case "always":
			parts = append(parts, "--build")
		case "never":
			parts = append(parts, "--no-build")
		}
	}
	switch v := args["pull"].(type) {
	case bool:
		if v {
			parts = append(parts, "--pull", "always")
		}
	case string:
		switch v {
		case "always", "missing", "never":
			parts = append(parts, "--pull", v)
		}
	}
	switch fmt.Sprint(args["recreate"]) {
	case "always":
		parts = append(parts, "--force-recreate")
	case "never":
		parts = append(parts, "--no-recreate")
	}
	if getBoolArg(args, "force_recreate", false) {
		parts = append(parts, "--force-recreate")
	}
	if getBoolArg(args, "remove_orphans", false) {
		parts = append(parts, "--remove-orphans")
	}
	if getBoolArg(args, "wait", false) {
		parts = append(parts, "--wait")
		if t := getIntArg(args, "wait_timeout", 0); t > 0 {
			parts = append(parts, "--wait-timeout", fmt.Sprint(t))
		}
	}
	return strings.Join(parts, " ") + servicesArg(args)
}

func downCommand(args map[string]interface{}) string {
	parts := []string{"down"}
	if getBoolArg(args, "remove_volumes", false) {
		parts = append(parts, "-v")
	}
	if getBoolArg(args, "remove_orphans", false) {
		parts = append(parts, "--remove-orphans")
	}
	return strings.Join(parts, " ")
}

// composeV2Args reads the arguments of community.docker.docker_compose_v2:
// project_src for project_dir; files, profiles and env_files are read by
// newComposeProject; build, pull and recreate stay policies
func composeV2Args(args map[string]interface{}) {
	if _, ok := args["project_dir"]; !ok {
		if src, ok := args["project_src"].(string); ok {
			args["project_dir"] = src
		}
	}
}
