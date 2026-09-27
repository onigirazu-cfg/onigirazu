package modules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAptUpgradeArg(t *testing.T) {
	for in, want := range map[interface{}]string{"yes": "upgrade", true: "upgrade", "safe": "upgrade", "dist": "dist-upgrade", "full": "dist-upgrade", "no": "", false: ""} {
		got, err := aptUpgradeArg(map[string]interface{}{"upgrade": in})
		assert.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	_, err := aptUpgradeArg(map[string]interface{}{"upgrade": "everything"})
	assert.Error(t, err)
}

func TestAptChanged(t *testing.T) {
	assert.False(t, aptChanged("0 upgraded, 0 newly installed, 0 to remove and 3 not upgraded."))
	assert.True(t, aptChanged("2 upgraded, 0 newly installed, 0 to remove and 0 not upgraded."))
	assert.True(t, aptChanged("0 upgraded, 0 newly installed, 1 to remove and 0 not upgraded."))
	assert.False(t, aptChanged("Reading package lists..."))
}

func TestSetFactCacheableIsAnOption(t *testing.T) {
	assert.Equal(t, map[string]interface{}{"x": 1}, factArgs(map[string]interface{}{"x": 1, "cacheable": true, "_vars": nil}))
}

func TestSystemdDaemonReloadWithoutName(t *testing.T) {
	m := NewSystemdModule()
	assert.NoError(t, m.Validate(map[string]interface{}{"daemon_reload": true}))
	assert.Error(t, m.Validate(map[string]interface{}{"state": "started"}))
}
