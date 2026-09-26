package rollback

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestResourceFromResult(t *testing.T) {
	file := types.TaskResult{TaskName: "conf", Module: "copy", Before: map[string]interface{}{
		"path": "/etc/app", "kind": "file", "content": "v1\n", "mode": "0644", "owner": "root", "group": "root", "_become": true}}
	r, ok := ResourceFromResult(file, "web1", 7)
	assert.True(t, ok)
	assert.True(t, r.Reversible)
	assert.Equal(t, "copy", r.RollbackOp.Module)
	assert.Equal(t, 7, r.RollbackOp.Order)
	assert.Equal(t, map[string]interface{}{"dest": "/etc/app", "content": "v1\n", "mode": "0644", "owner": "root", "group": "root", "_become": true}, r.RollbackOp.Args)
	assert.Equal(t, "restore content (3 bytes) mode=0644 owner=root group=root", describeOperation(r.RollbackOp))

	absent, _ := ResourceFromResult(types.TaskResult{Before: map[string]interface{}{"path": "/tmp/new", "kind": "absent"}}, "web1", 1)
	assert.Equal(t, map[string]interface{}{"path": "/tmp/new", "state": "absent"}, absent.RollbackOp.Args)
	assert.Equal(t, "remove (did not exist before)", describeOperation(absent.RollbackOp))

	binary, ok := ResourceFromResult(types.TaskResult{Before: map[string]interface{}{"path": "/bin/x", "kind": "file"}}, "web1", 1)
	assert.True(t, ok)
	assert.False(t, binary.Reversible)
	assert.Nil(t, binary.RollbackOp)

	_, ok = ResourceFromResult(types.TaskResult{Module: "apt"}, "web1", 1)
	assert.False(t, ok)
}
