package cli

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestWriteRunResult(t *testing.T) {
	result := &types.PlaybookResult{
		Failed: true,
		Plays: []types.PlayResult{{Name: "p", Hosts: []types.HostResult{{
			Host: "web1", Failed: true,
			Tasks: []types.TaskResult{{TaskName: "t", Failed: true, Error: "boom"}},
		}}}},
	}

	var out bytes.Buffer
	writeRunResult(&out, "json", result, "site.yml", time.Now())
	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal(out.Bytes(), &doc), "stdout is one JSON document")
	assert.Equal(t, "failed", doc["status"])
	assert.NotContains(t, doc, "playbook_result")

	out.Reset()
	writeRunResult(&out, "yaml", result, "site.yml", time.Now())
	var ydoc map[string]interface{}
	require.NoError(t, yaml.Unmarshal(out.Bytes(), &ydoc))
	assert.Equal(t, "failed", ydoc["status"])
	assert.Contains(t, ydoc, "total_failed", "YAML keys are the JSON ones")
}
