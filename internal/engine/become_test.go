package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestEffectiveBecome(t *testing.T) {
	play := becomeSettings{Become: true, User: "root", Method: "sudo"}

	assert.Equal(t, play, effectiveBecome(&types.Task{}, play), "task inherits the play")
	assert.Equal(t, becomeSettings{Become: true, User: "postgres", Method: "sudo"},
		effectiveBecome(&types.Task{BecomeUser: "postgres"}, play), "task user wins")
	assert.Equal(t, becomeSettings{}, effectiveBecome(&types.Task{}, becomeSettings{}), "no become anywhere")
	assert.Equal(t, becomeSettings{Become: true, User: "app"},
		effectiveBecome(&types.Task{Become: true, BecomeUser: "app"}, becomeSettings{}), "task-level become")
}
