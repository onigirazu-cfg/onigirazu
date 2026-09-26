package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerialBatches(t *testing.T) {
	tests := []struct {
		serial interface{}
		total  int
		want   []int
	}{
		{nil, 5, []int{5}},
		{1, 3, []int{1, 1, 1}},
		{2, 5, []int{2, 2, 1}},
		{"2", 4, []int{2, 2}},
		{"40%", 5, []int{2, 2, 1}},
		{"10%", 5, []int{1, 1, 1, 1, 1}},
		{[]interface{}{1, 5, "50%"}, 10, []int{1, 5, 4}},
		{[]interface{}{1, "50%"}, 10, []int{1, 5, 4}},
		{0, 4, []int{4}},
		{10, 3, []int{3}},
	}
	for _, tt := range tests {
		got, err := serialBatches(tt.serial, tt.total)
		require.NoError(t, err, tt.serial)
		assert.Equal(t, tt.want, got, tt.serial)
	}
	_, err := serialBatches("many", 3)
	assert.Error(t, err)
}
