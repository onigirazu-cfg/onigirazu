package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type DockerContainerModule struct {
	BaseModule
}

type ContainerState struct {
	Name          string            `json:"name"`
	ID            string            `json:"id"`
	Image         string            `json:"image"`
	Status        string            `json:"status"`
	State         string            `json:"state"`
	Running       bool              `json:"running"`
	Ports         map[string]string `json:"ports"`
	Volumes       []string          `json:"volumes"`
	Env           []string          `json:"env"`
	Networks      []string          `json:"networks"`
	RestartPolicy string            `json:"restart_policy"`
	NanoCPUs      int64             `json:"nano_cpus"`
	Memory        int64             `json:"memory"`
	MemorySwap    int64             `json:"memory_swap"`
	inspect       map[string]interface{}
}

func NewDockerContainerModule() *DockerContainerModule {
	return &DockerContainerModule{
		BaseModule: BaseModule{
			name:        "docker_container",
			description: "Manage Docker containers",
		},
	}
}

func (m *DockerContainerModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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

	name, ok := args["name"].(string)
	if !ok || name == "" {
		result.Success = false
		result.Error = "container name is required"
		return result, fmt.Errorf("container name is required")
	}

	state, _ := args["state"].(string)
	if state == "" {
		state = "started"
	}

	currentState, exists, err := m.getContainerState(ctx, exec, name)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to get container state: %v", err)
		return result, err
	}

	limits, err := containerLimits(args)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, err
	}
	var update []string
	if exists && state != "absent" {
		update = limits.updateArgs(currentState)
	}

	// an existing container whose options differ from the task is created
	// again, as Ansible's docker_container does (recreate: true always)
	var diffs []string
	if exists && (state == "present" || state == "started") {
		imageID := ""
		if image := getStringArg(args, "image", ""); image != "" {
			out, err := exec.Execute(shellJoin("docker", "image", "inspect", "-f", "{{.Id}}", image) + " 2>/dev/null")
			if err == nil {
				imageID = strings.TrimSpace(out)
			}
		}
		diffs = containerDiffs(currentState.inspect, args, imageID)
		if getBoolArg(args, "recreate", false) {
			diffs = append(diffs, "recreate")
		}
		if len(diffs) > 0 {
			update = nil // the new container gets the limits from its creation
		}
	}

	if inCheckMode(args) {
		running := exists && currentState.Running
		if action := plannedContainerAction(state, exists, running); action != "" {
			result.Changed = true
			result.Output["action"] = action
		}
		if len(diffs) > 0 {
			result.Changed = true
			result.Output["action"] = "recreated"
			result.Output["differences"] = diffs
		}
		if len(update) > 0 {
			result.Changed = true
			result.Output["updated"] = true
		}
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// cpu and memory limits change in place, the way docker_container does it
	if len(update) > 0 {
		cmd := shellJoin(append(append([]string{"docker", "update"}, update...), name)...)
		if _, err := exec.Execute(cmd); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to update container limits: %v", err)
			return result, err
		}
		result.Changed = true
		result.Output["updated"] = true
	}

	if len(diffs) > 0 {
		if err := m.removeContainer(ctx, exec, name, map[string]interface{}{"force": true}); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to remove container to recreate it: %v", err)
			return result, err
		}
		exists = false
		result.Output["differences"] = diffs
	}

	switch state {
	case "present", "started":
		if !exists {
			if err := m.createContainer(ctx, exec, name, args); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to create container: %v", err)
				return result, err
			}
			result.Changed = true
			result.Output["action"] = map[bool]string{true: "recreated", false: "created"}[len(diffs) > 0]
		}

		if state == "started" && (!exists || !currentState.Running) {
			if err := m.startContainer(ctx, exec, name); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to start container: %v", err)
				return result, err
			}
			result.Changed = true
			result.Output["action"] = "started"
		}

	case "stopped":
		if exists && currentState.Running {
			if err := m.stopContainer(ctx, exec, name); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to stop container: %v", err)
				return result, err
			}
			result.Changed = true
			result.Output["action"] = "stopped"
		}

	case "restarted":
		if exists {
			if err := m.restartContainer(ctx, exec, name); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to restart container: %v", err)
				return result, err
			}
			result.Changed = true
			result.Output["action"] = "restarted"
		}

	case "absent":
		if exists {
			if currentState.Running {
				if err := m.stopContainer(ctx, exec, name); err != nil {
					result.Success = false
					result.Error = fmt.Sprintf("failed to stop container: %v", err)
					return result, err
				}
			}
			if err := m.removeContainer(ctx, exec, name, args); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to remove container: %v", err)
				return result, err
			}
			result.Changed = true
			result.Output["action"] = "removed"
		}
	}

	finalState, _, _ := m.getContainerState(ctx, exec, name)
	result.Output["container"] = finalState
	result.Duration = time.Since(startTime)

	return result, nil
}

func (m *DockerContainerModule) getContainerState(ctx context.Context, exec *executor.CommandExecutor, name string) (*ContainerState, bool, error) {
	cmd := shellJoin("docker", "inspect", name) + " 2>/dev/null"
	stdout, err := exec.Execute(cmd)
	if err != nil {
		return nil, false, nil
	}

	var containers []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &containers); err != nil {
		return nil, false, err
	}

	if len(containers) == 0 {
		return nil, false, nil
	}

	container := containers[0]
	id, _ := container["Id"].(string)
	if len(id) > 12 {
		id = id[:12]
	}
	containerConfig, _ := container["Config"].(map[string]interface{})
	image, _ := containerConfig["Image"].(string)
	state := &ContainerState{
		Name:    name,
		ID:      id,
		Image:   image,
		inspect: container,
	}

	if stateMap, ok := container["State"].(map[string]interface{}); ok {
		state.Running, _ = stateMap["Running"].(bool)
		state.Status, _ = stateMap["Status"].(string)
		state.State = state.Status
	}
	if hostConfig, ok := container["HostConfig"].(map[string]interface{}); ok {
		if v, ok := hostConfig["NanoCpus"].(float64); ok {
			state.NanoCPUs = int64(v)
		}
		if v, ok := hostConfig["Memory"].(float64); ok {
			state.Memory = int64(v)
		}
		if v, ok := hostConfig["MemorySwap"].(float64); ok {
			state.MemorySwap = int64(v)
		}
	}

	return state, true, nil
}

func (m *DockerContainerModule) createContainer(ctx context.Context, exec *executor.CommandExecutor, name string, args map[string]interface{}) error {
	image, _ := args["image"].(string)
	if image == "" {
		return fmt.Errorf("image is required to create container")
	}

	cmdParts := []string{"docker", "run", "-d", "--name", name}

	if ports, ok := args["ports"].([]interface{}); ok {
		for _, port := range ports {
			cmdParts = append(cmdParts, "-p", fmt.Sprintf("%v", port))
		}
	}

	if volumes, ok := args["volumes"].([]interface{}); ok {
		for _, vol := range volumes {
			cmdParts = append(cmdParts, "-v", fmt.Sprintf("%v", vol))
		}
	}

	if env, ok := args["env"].(map[string]interface{}); ok {
		for k, v := range env {
			cmdParts = append(cmdParts, "-e", fmt.Sprintf("%s=%v", k, v))
		}
	}

	if networks, ok := args["networks"].([]interface{}); ok {
		for _, net := range networks {
			// Ansible lists networks as {name: ...}
			if m, ok := net.(map[string]interface{}); ok {
				net = m["name"]
			}
			cmdParts = append(cmdParts, "--network", fmt.Sprintf("%v", net))
		}
	}

	if restart, ok := args["restart_policy"].(string); ok {
		cmdParts = append(cmdParts, "--restart", restart)
	}

	limits, err := containerLimits(args)
	if err != nil {
		return err
	}
	cmdParts = append(cmdParts, limits.runArgs()...)

	cmdParts = append(cmdParts, image)
	// the command is split into words like a shell would, then each is quoted
	switch command := args["command"].(type) {
	case string:
		if command != "" {
			words, err := splitCommandLine(command)
			if err != nil {
				return fmt.Errorf("invalid command: %w", err)
			}
			cmdParts = append(cmdParts, words...)
		}
	case []interface{}:
		// a list is the argv as it is, as in Ansible
		for _, w := range command {
			cmdParts = append(cmdParts, fmt.Sprint(w))
		}
	case []string:
		cmdParts = append(cmdParts, command...)
	}

	cmd := shellJoin(cmdParts...)
	if _, err := exec.Execute(cmd); err != nil {
		return fmt.Errorf("failed to create container: %s", err.Error())
	}

	return nil
}

func (m *DockerContainerModule) startContainer(ctx context.Context, exec *executor.CommandExecutor, name string) error {
	cmd := shellJoin("docker", "start", name)
	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to start container: %s", err.Error())
	}
	return nil
}

func (m *DockerContainerModule) stopContainer(ctx context.Context, exec *executor.CommandExecutor, name string) error {
	cmd := shellJoin("docker", "stop", name)
	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to stop container: %s", err.Error())
	}
	return nil
}

func (m *DockerContainerModule) restartContainer(ctx context.Context, exec *executor.CommandExecutor, name string) error {
	cmd := shellJoin("docker", "restart", name)
	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to restart container: %s", err.Error())
	}
	return nil
}

func (m *DockerContainerModule) removeContainer(ctx context.Context, exec *executor.CommandExecutor, name string, args map[string]interface{}) error {
	force := getBoolArg(args, "force", false)
	cmdParts := []string{"docker", "rm"}
	if force {
		cmdParts = append(cmdParts, "-f")
	}
	cmdParts = append(cmdParts, name)

	cmd := shellJoin(cmdParts...)
	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to remove container: %s", err.Error())
	}
	return nil
}

// Validate validates docker_container module arguments
func (m *DockerContainerModule) Validate(args map[string]interface{}) error {
	return requireStringArg(args, "name")
}

// plannedContainerAction is what a container task would do, for check mode
func plannedContainerAction(state string, exists, running bool) string {
	switch state {
	case "present":
		if !exists {
			return "created"
		}
	case "started":
		if !exists || !running {
			return "started"
		}
	case "stopped":
		if exists && running {
			return "stopped"
		}
	case "restarted":
		if exists {
			return "restarted"
		}
	case "absent":
		if exists {
			return "removed"
		}
	}
	return ""
}
