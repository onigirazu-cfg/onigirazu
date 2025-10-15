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

type PodmanModule struct {
	BaseModule
	executor *executor.CommandExecutor
}

type PodmanContainerState struct {
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
}

func NewPodmanModule() *PodmanModule {
	return &PodmanModule{
		BaseModule: BaseModule{
			name:        "podman",
			description: "Manage Podman containers",
		},
	}
}

func (m *PodmanModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  "podman",
		Host:      host.Name,
		Module:    m.GetName(),
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}

	if m.executor == nil {
		var err error
		m.executor, err = executor.NewCommandExecutor(host)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to create executor: %v", err)
			return result, err
		}
	}

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

	currentState, exists, err := m.getContainerState(ctx, name)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to get container state: %v", err)
		return result, err
	}

	switch state {
	case "present", "started":
		if !exists {
			if err := m.createContainer(ctx, name, args); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to create container: %v", err)
				return result, err
			}
			result.Changed = true
			result.Output["action"] = "created"
		}

		if state == "started" && (!exists || !currentState.Running) {
			if err := m.startContainer(ctx, name); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to start container: %v", err)
				return result, err
			}
			result.Changed = true
			result.Output["action"] = "started"
		}

	case "stopped":
		if exists && currentState.Running {
			if err := m.stopContainer(ctx, name); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to stop container: %v", err)
				return result, err
			}
			result.Changed = true
			result.Output["action"] = "stopped"
		}

	case "restarted":
		if exists {
			if err := m.restartContainer(ctx, name); err != nil {
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
				if err := m.stopContainer(ctx, name); err != nil {
					result.Success = false
					result.Error = fmt.Sprintf("failed to stop container: %v", err)
					return result, err
				}
			}
			if err := m.removeContainer(ctx, name, args); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to remove container: %v", err)
				return result, err
			}
			result.Changed = true
			result.Output["action"] = "removed"
		}
	}

	finalState, _, _ := m.getContainerState(ctx, name)
	result.Output["container"] = finalState
	result.Duration = time.Since(startTime)

	return result, nil
}

func (m *PodmanModule) getContainerState(ctx context.Context, name string) (*PodmanContainerState, bool, error) {
	cmd := fmt.Sprintf("podman inspect %s 2>/dev/null", name)
	stdout, err := m.executor.Execute(cmd)
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
	state := &PodmanContainerState{
		Name:  name,
		ID:    container["Id"].(string)[:12],
		Image: container["Config"].(map[string]interface{})["Image"].(string),
	}

	if stateMap, ok := container["State"].(map[string]interface{}); ok {
		state.Running = stateMap["Running"].(bool)
		state.Status = stateMap["Status"].(string)
		state.State = state.Status
	}

	return state, true, nil
}

func (m *PodmanModule) createContainer(ctx context.Context, name string, args map[string]interface{}) error {
	image, _ := args["image"].(string)
	if image == "" {
		return fmt.Errorf("image is required to create container")
	}

	cmdParts := []string{"podman run -d --name", name}

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
			cmdParts = append(cmdParts, "--network", fmt.Sprintf("%v", net))
		}
	}

	if restart, ok := args["restart_policy"].(string); ok {
		cmdParts = append(cmdParts, "--restart", restart)
	}

	if rootless, ok := args["rootless"].(bool); ok && rootless {
		cmdParts = append(cmdParts, "--userns=keep-id")
	}

	if command, ok := args["command"].(string); ok {
		cmdParts = append(cmdParts, image, command)
	} else {
		cmdParts = append(cmdParts, image)
	}

	cmd := strings.Join(cmdParts, " ")
	_, err := m.executor.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to create container: %s", err.Error())
	}

	return nil
}

func (m *PodmanModule) startContainer(ctx context.Context, name string) error {
	cmd := fmt.Sprintf("podman start %s", name)
	_, err := m.executor.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to start container: %s", err.Error())
	}
	return nil
}

func (m *PodmanModule) stopContainer(ctx context.Context, name string) error {
	cmd := fmt.Sprintf("podman stop %s", name)
	_, err := m.executor.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to stop container: %s", err.Error())
	}
	return nil
}

func (m *PodmanModule) restartContainer(ctx context.Context, name string) error {
	cmd := fmt.Sprintf("podman restart %s", name)
	_, err := m.executor.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to restart container: %s", err.Error())
	}
	return nil
}

func (m *PodmanModule) removeContainer(ctx context.Context, name string, args map[string]interface{}) error {
	force, _ := args["force"].(bool)
	cmdParts := []string{"podman rm"}
	if force {
		cmdParts = append(cmdParts, "-f")
	}
	cmdParts = append(cmdParts, name)

	cmd := strings.Join(cmdParts, " ")
	_, err := m.executor.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to remove container: %s", err.Error())
	}
	return nil
}
