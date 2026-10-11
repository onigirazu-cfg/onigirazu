package logger

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedact(t *testing.T) {
	defer resetRedaction()
	Redact("s3cr3t-value")
	Redact("s3cr3t")
	Redact("ab") // too short to mask
	assert.Equal(t, "pw=*** and *** ab", Scrub("pw=s3cr3t-value and s3cr3t ab"))

	var buf bytes.Buffer
	l := NewWithWriter(true, &buf)
	l.Debug("vars: map[ansible_password:%s]", "s3cr3t-value")
	assert.Contains(t, buf.String(), "ansible_password:***")
	assert.NotContains(t, buf.String(), "s3cr3t")
}
