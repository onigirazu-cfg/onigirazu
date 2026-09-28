package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// DockerHostInfoModule reports the Docker host and, on request, its
// containers (community.docker.docker_host_info): containers,
// containers_all, containers_filters (name, label, status, ...). A
// container is shaped as the Docker API lists it: Id, Names (with the
// leading /), Image, State, Status, Labels.
type DockerHostInfoModule struct {
	*BaseModule
}

// NewDockerHostInfoModule creates the docker_host_info module
func NewDockerHostInfoModule() *DockerHostInfoModule {
	return &DockerHostInfoModule{BaseModule: NewBaseModule("docker_host_info")}
}

func (m *DockerHostInfoModule) GetDescription() string { return "Information about the Docker host" }

func (m *DockerHostInfoModule) Validate(map[string]interface{}) error { return nil }

func (m *DockerHostInfoModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name,
		Timestamp: start, Success: true, Output: map[string]interface{}{}}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error, result.Duration = false, msg, time.Since(start)
		return result, nil
	}
	info, err := runShellOnHost(ctx, host, args, "docker info --format '{{json .}}'")
	if err != nil {
		return fail(fmt.Sprintf("docker info failed: %v", err))
	}
	var hostInfo map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(info)), &hostInfo); err == nil {
		result.Output["host_info"] = hostInfo
	}
	result.Output["can_talk_to_docker"] = true

	if getBoolArg(args, "containers", false) {
		cmd := "docker ps --no-trunc --format '{{json .}}'"
		if getBoolArg(args, "containers_all", false) {
			cmd += " -a"
		}
		if filters, ok := args["containers_filters"].(map[string]interface{}); ok {
			for key, v := range filters {
				values, isList := v.([]interface{})
				if !isList {
					values = []interface{}{v}
				}
				for _, one := range values {
					cmd += " --filter " + shellQuote(key+"="+fmt.Sprint(one))
				}
			}
		}
		out, err := runShellOnHost(ctx, host, args, cmd)
		if err != nil {
			return fail(fmt.Sprintf("docker ps failed: %v", err))
		}
		containers := []interface{}{}
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			var c map[string]interface{}
			if err := json.Unmarshal([]byte(line), &c); err != nil {
				continue
			}
			var names []interface{}
			for _, n := range strings.Split(fmt.Sprint(c["Names"]), ",") {
				names = append(names, "/"+n)
			}
			containers = append(containers, map[string]interface{}{
				"Id": c["ID"], "Names": names, "Image": c["Image"], "State": c["State"],
				"Status": c["Status"], "Labels": c["Labels"], "Command": c["Command"],
			})
		}
		result.Output["containers"] = containers
	}
	result.Duration = time.Since(start)
	return result, nil
}
