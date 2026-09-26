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
		TaskName:  "docker_compose",
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

	projectDir, ok := args["project_dir"].(string)
	if !ok || projectDir == "" {
		result.Success = false
		result.Error = "project_dir is required"
		return result, fmt.Errorf("project_dir is required")
	}

	state, _ := args["state"].(string)
	if state == "" {
		state = "present"
	}

	composeFile, _ := args["file"].(string)

	projectName, _ := args["project_name"].(string)

	switch state {
	case "present":
		before := m.snapshot(exec, projectDir, composeFile, projectName)
		if err := m.composeUp(ctx, exec, projectDir, composeFile, projectName, args); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to start compose: %v", err)
			return result, err
		}
		result.Changed = m.snapshot(exec, projectDir, composeFile, projectName) != before
		result.Output["action"] = "started"

	case "absent":
		before := m.snapshot(exec, projectDir, composeFile, projectName)
		if err := m.composeDown(ctx, exec, projectDir, composeFile, projectName, args); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to stop compose: %v", err)
			return result, err
		}
		result.Changed = m.snapshot(exec, projectDir, composeFile, projectName) != before
		result.Output["action"] = "stopped"

	case "restarted":
		if err := m.composeRestart(ctx, exec, projectDir, composeFile, projectName, args); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to restart compose: %v", err)
			return result, err
		}
		result.Changed = true
		result.Output["action"] = "restarted"

	case "pull":
		if err := m.composePull(ctx, exec, projectDir, composeFile, projectName); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to pull images: %v", err)
			return result, err
		}
		result.Changed = true
		result.Output["action"] = "pulled"

	case "build":
		if err := m.composeBuild(ctx, exec, projectDir, composeFile, projectName, args); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to build: %v", err)
			return result, err
		}
		result.Changed = true
		result.Output["action"] = "built"
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// buildComposeCmd prefers the Compose v2 plugin and falls back to the v1
// docker-compose binary. The file is passed only when set explicitly, so
// compose.yaml and docker-compose.yml are both found by default.
func (m *DockerComposeModule) buildComposeCmd(projectDir, composeFile, projectName string, baseCmd string) string {
	parts := []string{"c"}
	if composeFile != "" {
		parts = append(parts, "-f", shellQuote(composeFile))
	}
	if projectName != "" {
		parts = append(parts, "-p", shellQuote(projectName))
	}
	parts = append(parts, baseCmd)
	script := `c() { if docker compose version >/dev/null 2>&1; then docker compose "$@"; else docker-compose "$@"; fi; }; ` +
		"cd " + shellQuote(projectDir) + " && " + strings.Join(parts, " ")
	return "sh -c " + shellQuote(script)
}

// snapshot lists all project containers and the running ones; comparing it
// before and after tells whether up/down changed anything
func (m *DockerComposeModule) snapshot(exec *executor.CommandExecutor, projectDir, composeFile, projectName string) string {
	out, err := exec.Execute(m.buildComposeCmd(projectDir, composeFile, projectName, "ps -a -q; echo --; c "+composeArgs(composeFile, projectName)+"ps -q"))
	if err != nil {
		return "error: " + err.Error()
	}
	return out
}

func composeArgs(composeFile, projectName string) string {
	s := ""
	if composeFile != "" {
		s += "-f " + shellQuote(composeFile) + " "
	}
	if projectName != "" {
		s += "-p " + shellQuote(projectName) + " "
	}
	return s
}

func (m *DockerComposeModule) composeUp(ctx context.Context, exec *executor.CommandExecutor, projectDir, composeFile, projectName string, args map[string]interface{}) error {
	cmdParts := []string{}

	detach := getBoolArg(args, "detach", false)
	if detach || args["detach"] == nil {
		cmdParts = append(cmdParts, "up -d")
	} else {
		cmdParts = append(cmdParts, "up")
	}

	build := getBoolArg(args, "build", false)
	if build {
		cmdParts = append(cmdParts, "--build")
	}

	forceRecreate := getBoolArg(args, "force_recreate", false)
	if forceRecreate {
		cmdParts = append(cmdParts, "--force-recreate")
	}

	if services, ok := args["services"].([]interface{}); ok {
		for _, svc := range services {
			cmdParts = append(cmdParts, fmt.Sprintf("%v", svc))
		}
	}

	cmd := m.buildComposeCmd(projectDir, composeFile, projectName, strings.Join(cmdParts, " "))
	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("compose up failed: %s", err.Error())
	}

	return nil
}

func (m *DockerComposeModule) composeDown(ctx context.Context, exec *executor.CommandExecutor, projectDir, composeFile, projectName string, args map[string]interface{}) error {
	cmdParts := []string{"down"}

	removeVolumes := getBoolArg(args, "remove_volumes", false)
	if removeVolumes {
		cmdParts = append(cmdParts, "-v")
	}

	removeOrphans := getBoolArg(args, "remove_orphans", false)
	if removeOrphans {
		cmdParts = append(cmdParts, "--remove-orphans")
	}

	cmd := m.buildComposeCmd(projectDir, composeFile, projectName, strings.Join(cmdParts, " "))
	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("compose down failed: %s", err.Error())
	}

	return nil
}

func (m *DockerComposeModule) composeRestart(ctx context.Context, exec *executor.CommandExecutor, projectDir, composeFile, projectName string, args map[string]interface{}) error {
	cmdParts := []string{"restart"}

	if services, ok := args["services"].([]interface{}); ok {
		for _, svc := range services {
			cmdParts = append(cmdParts, fmt.Sprintf("%v", svc))
		}
	}

	cmd := m.buildComposeCmd(projectDir, composeFile, projectName, strings.Join(cmdParts, " "))
	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("compose restart failed: %s", err.Error())
	}

	return nil
}

func (m *DockerComposeModule) composePull(ctx context.Context, exec *executor.CommandExecutor, projectDir, composeFile, projectName string) error {
	cmd := m.buildComposeCmd(projectDir, composeFile, projectName, "pull")
	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("compose pull failed: %s", err.Error())
	}

	return nil
}

func (m *DockerComposeModule) composeBuild(ctx context.Context, exec *executor.CommandExecutor, projectDir, composeFile, projectName string, args map[string]interface{}) error {
	cmdParts := []string{"build"}

	noCache := getBoolArg(args, "nocache", false)
	if noCache {
		cmdParts = append(cmdParts, "--no-cache")
	}

	pull := getBoolArg(args, "pull", false)
	if pull {
		cmdParts = append(cmdParts, "--pull")
	}

	if services, ok := args["services"].([]interface{}); ok {
		for _, svc := range services {
			cmdParts = append(cmdParts, fmt.Sprintf("%v", svc))
		}
	}

	cmd := m.buildComposeCmd(projectDir, composeFile, projectName, strings.Join(cmdParts, " "))
	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("compose build failed: %s", err.Error())
	}

	return nil
}
