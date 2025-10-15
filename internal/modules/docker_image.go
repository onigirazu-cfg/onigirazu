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

type DockerImageModule struct {
	BaseModule
}

type ImageInfo struct {
	ID          string    `json:"id"`
	Repository  string    `json:"repository"`
	Tag         string    `json:"tag"`
	Digest      string    `json:"digest"`
	Created     time.Time `json:"created"`
	Size        string    `json:"size"`
	VirtualSize int64     `json:"virtual_size"`
}

func NewDockerImageModule() *DockerImageModule {
	return &DockerImageModule{
		BaseModule: BaseModule{
			name:        "docker_image",
			description: "Manage Docker images",
		},
	}
}

func (m *DockerImageModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  "docker_image",
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
		result.Error = "image name is required"
		return result, fmt.Errorf("image name is required")
	}

	state, _ := args["state"].(string)
	if state == "" {
		state = "present"
	}

	tag, _ := args["tag"].(string)
	if tag == "" {
		tag = "latest"
	}

	fullName := fmt.Sprintf("%s:%s", name, tag)
	exists, _, err := m.imageExists(ctx, exec, fullName)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to check image: %v", err)
		return result, err
	}

	switch state {
	case "present":
		if !exists {
			if err := m.pullImage(ctx, exec, fullName, args); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to pull image: %v", err)
				return result, err
			}
			result.Changed = true
			result.Output["action"] = "pulled"
		} else {
			force, _ := args["force"].(bool)
			if force {
				if err := m.pullImage(ctx, exec, fullName, args); err != nil {
					result.Success = false
					result.Error = fmt.Sprintf("failed to pull image: %v", err)
					return result, err
				}
				result.Changed = true
				result.Output["action"] = "updated"
			}
		}

	case "absent":
		if exists {
			if err := m.removeImage(ctx, exec, fullName, args); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to remove image: %v", err)
				return result, err
			}
			result.Changed = true
			result.Output["action"] = "removed"
		}

	case "build":
		if err := m.buildImage(ctx, exec, name, tag, args); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to build image: %v", err)
			return result, err
		}
		result.Changed = true
		result.Output["action"] = "built"
	}

	if state != "absent" {
		_, imgInfo, _ := m.imageExists(ctx, exec, fullName)
		result.Output["image"] = imgInfo
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *DockerImageModule) imageExists(ctx context.Context, exec *executor.CommandExecutor, name string) (bool, *ImageInfo, error) {
	cmd := fmt.Sprintf("docker images --format '{{json .}}' %s", name)
	stdout, err := exec.Execute(cmd)
	if err != nil {
		return false, nil, nil
	}

	if strings.TrimSpace(stdout) == "" {
		return false, nil, nil
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) == 0 {
		return false, nil, nil
	}

	var imageData map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &imageData); err != nil {
		return false, nil, err
	}

	info := &ImageInfo{
		ID:         imageData["ID"].(string),
		Repository: imageData["Repository"].(string),
		Tag:        imageData["Tag"].(string),
		Size:       imageData["Size"].(string),
	}

	return true, info, nil
}

func (m *DockerImageModule) pullImage(ctx context.Context, exec *executor.CommandExecutor, name string, args map[string]interface{}) error {
	cmdParts := []string{"docker pull"}

	if platform, ok := args["platform"].(string); ok {
		cmdParts = append(cmdParts, "--platform", platform)
	}

	cmdParts = append(cmdParts, name)
	cmd := strings.Join(cmdParts, " ")

	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to pull image: %s", err.Error())
	}

	return nil
}

func (m *DockerImageModule) removeImage(ctx context.Context, exec *executor.CommandExecutor, name string, args map[string]interface{}) error {
	cmdParts := []string{"docker rmi"}

	force, _ := args["force"].(bool)
	if force {
		cmdParts = append(cmdParts, "-f")
	}

	cmdParts = append(cmdParts, name)
	cmd := strings.Join(cmdParts, " ")

	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to remove image: %s", err.Error())
	}

	return nil
}

func (m *DockerImageModule) buildImage(ctx context.Context, exec *executor.CommandExecutor, name, tag string, args map[string]interface{}) error {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("path is required for building image")
	}

	cmdParts := []string{"docker build"}

	if dockerfile, ok := args["dockerfile"].(string); ok {
		cmdParts = append(cmdParts, "-f", dockerfile)
	}

	if buildArgs, ok := args["build_args"].(map[string]interface{}); ok {
		for k, v := range buildArgs {
			cmdParts = append(cmdParts, "--build-arg", fmt.Sprintf("%s=%v", k, v))
		}
	}

	if nocache, ok := args["nocache"].(bool); ok && nocache {
		cmdParts = append(cmdParts, "--no-cache")
	}

	if pull, ok := args["pull"].(bool); ok && pull {
		cmdParts = append(cmdParts, "--pull")
	}

	fullName := fmt.Sprintf("%s:%s", name, tag)
	cmdParts = append(cmdParts, "-t", fullName, path)

	cmd := strings.Join(cmdParts, " ")
	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to build image: %s", err.Error())
	}

	return nil
}
