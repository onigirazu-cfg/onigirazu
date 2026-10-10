// Package tfstate reads Terraform/OpenTofu outputs for the tf_output lookup
// (a package of its own so the expression and inventory packages can share it)
package tfstate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// TerraformOutputs reads the outputs of a project (`terraform output
// -json`) or of a state file; the tf_output lookup uses it
func Outputs(ctx context.Context, project, stateFile, binary string) (map[string]interface{}, error) {
	if binary == "" {
		binary = "terraform"
	}
	var raw map[string]struct {
		Value interface{} `json:"value"`
	}
	switch {
	case stateFile != "":
		data, err := os.ReadFile(stateFile) // #nosec G304 -- the lookup names the file
		if err != nil {
			return nil, err
		}
		var state struct {
			Outputs map[string]struct {
				Value interface{} `json:"value"`
			} `json:"outputs"`
		}
		if err := json.Unmarshal(data, &state); err != nil {
			return nil, fmt.Errorf("%s: %w", stateFile, err)
		}
		raw = state.Outputs
	default:
		cmd := exec.CommandContext(ctx, binary, "output", "-json") // #nosec G204 -- the lookup's own binary
		cmd.Dir = project
		cmd.Env = append(os.Environ(), "TF_IN_AUTOMATION=1", "TF_CLI_ARGS=-no-color")
		out, err := cmd.Output()
		if err != nil {
			msg := err.Error()
			if ee, ok := err.(*exec.ExitError); ok {
				msg = strings.TrimSpace(string(ee.Stderr))
			}
			return nil, fmt.Errorf("%s output -json: %s", binary, msg)
		}
		if err := json.Unmarshal(out, &raw); err != nil {
			return nil, fmt.Errorf("%s output -json: %w", binary, err)
		}
	}
	outputs := make(map[string]interface{}, len(raw))
	for k, v := range raw {
		outputs[k] = v.Value
	}
	return outputs, nil
}
