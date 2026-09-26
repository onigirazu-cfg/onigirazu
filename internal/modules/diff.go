package modules

import (
	"bytes"
	"context"
	"fmt"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// --diff: modules that change files report what they change as
// Output["diff"], a list of {before_header, after_header, before, after}
// as in Ansible; the CLI renders them as unified diffs.

// maxDiffSize is the largest content shown in a diff
const maxDiffSize = 256 << 10

func diffRequested(args map[string]interface{}) bool {
	b, _ := args["_diff"].(bool)
	return b
}

// addDiff records one before/after pair when --diff is on
func addDiff(args map[string]interface{}, result *types.TaskResult, header, before, after string) {
	if !diffRequested(args) || before == after {
		return
	}
	if result.Output == nil {
		result.Output = map[string]interface{}{}
	}
	list, _ := result.Output["diff"].([]interface{})
	result.Output["diff"] = append(list, map[string]interface{}{
		"before_header": header, "after_header": header, "before": before, "after": after,
	})
}

// diffText is file content as a diff shows it: binary or large content is
// replaced by a note
func diffText(data []byte, exists bool) string {
	switch {
	case !exists:
		return ""
	case len(data) > maxDiffSize:
		return fmt.Sprintf("(%d bytes, too large to show)\n", len(data))
	case bytes.IndexByte(data, 0) >= 0:
		return fmt.Sprintf("(binary, %d bytes)\n", len(data))
	}
	return string(data)
}

// addFileDiff records the change of a file's content; the current content
// is read from the host only when --diff is on
func addFileDiff(ctx context.Context, host types.Host, args map[string]interface{}, result *types.TaskResult, path string, exists bool, after []byte) {
	if !diffRequested(args) {
		return
	}
	var before []byte
	if exists {
		data, ok, err := readHostFile(ctx, host, args, path)
		if err != nil {
			addDiff(args, result, path, fmt.Sprintf("(cannot read: %v)\n", err), diffText(after, true))
			return
		}
		before, exists = data, ok
	}
	addDiff(args, result, path, diffText(before, exists), diffText(after, true))
}
