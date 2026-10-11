package engine

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// an argument that still carries a variable's raw template fails the task
// with that variable's error; other arguments pass
func TestUnrenderedUsed(t *testing.T) {
	bad := map[string]unrenderedVar{"image": {text: "{{ (registry or 'x') }}/img", err: errors.New("interface conversion: string, not bool")}}
	name, err := unrenderedUsed(map[string]interface{}{"cmd": "docker pull {{ (registry or 'x') }}/img"}, bad)
	assert.Equal(t, "image", name)
	assert.ErrorContains(t, err, "not bool")

	name, err = unrenderedUsed(map[string]interface{}{"cmd": "echo fine", "env": map[string]interface{}{"A": []interface{}{"{{ (registry or 'x') }}/img"}}}, bad)
	assert.Equal(t, "image", name)
	assert.Error(t, err)

	_, err = unrenderedUsed(map[string]interface{}{"cmd": "echo fine"}, bad)
	assert.NoError(t, err)
	_, err = unrenderedUsed(map[string]interface{}{"cmd": "{{ (registry or 'x') }}"}, nil)
	assert.NoError(t, err)
}
