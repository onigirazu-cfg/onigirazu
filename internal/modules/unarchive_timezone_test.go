package modules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArchiveMembers(t *testing.T) {
	members := archiveMembers("./\n./app/\n./app/a.txt\napp/sub/\napp/sub/b.txt\n\n")
	assert.Equal(t, []string{"app", "app/a.txt", "app/sub", "app/sub/b.txt"}, members)
	assert.Equal(t, []string{"app"}, topLevel(members))
	assert.Equal(t, []string{"a", "b"}, topLevel([]string{"b/x", "a", "a/y"}))
}

func TestUnarchiveValidate(t *testing.T) {
	m := NewUnarchiveModule()
	assert.NoError(t, m.Validate(map[string]interface{}{"src": "a.tgz", "dest": "/opt"}))
	assert.Error(t, m.Validate(map[string]interface{}{"src": "a.tgz"}))
}

func TestTimezoneValidate(t *testing.T) {
	m := NewTimezoneModule()
	for _, ok := range []string{"UTC", "Europe/Madrid", "America/Argentina/Buenos_Aires", "Etc/GMT+3"} {
		assert.NoError(t, m.Validate(map[string]interface{}{"name": ok}), ok)
	}
	for _, bad := range []string{"", "../../etc/passwd", "Europe/Madrid; rm -rf /", "a b"} {
		assert.Error(t, m.Validate(map[string]interface{}{"name": bad}), bad)
	}
}
