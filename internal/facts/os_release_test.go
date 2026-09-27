package facts

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
)

func TestParseOSReleaseCodename(t *testing.T) {
	g := &Gatherer{}
	var ubuntu, rocky cache.SystemFacts
	g.parseOSRelease("ID=ubuntu\nVERSION_ID=\"24.04\"\nVERSION=\"24.04.1 LTS (Noble Numbat)\"\nVERSION_CODENAME=noble\n", &ubuntu)
	g.parseOSRelease("ID=\"rocky\"\nVERSION_ID=\"9.4\"\nVERSION=\"9.4 (Blue Onyx)\"\n", &rocky)
	assert.Equal(t, "noble", ubuntu.OSCodename)
	assert.Equal(t, "Blue Onyx", rocky.OSCodename)
}
