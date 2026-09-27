package modules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepoFilename(t *testing.T) {
	assert.Equal(t, "download_docker_com_linux_ubuntu",
		repoFilename("deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu noble stable"))
	assert.Equal(t, "dl_google_com_linux_chrome_deb", repoFilename("deb http://dl.google.com/linux/chrome/deb/ stable main"))
}

func TestRepoLineMatches(t *testing.T) {
	assert.True(t, repoLineMatches("deb http://x/ y main", normalizeRepoLine("deb   http://x/  y main")))
	assert.True(t, repoLineMatches("ppa:deadsnakes/ppa", "deb https://ppa.launchpadcontent.net/deadsnakes/ppa/ubuntu noble main"))
	assert.False(t, repoLineMatches("deb http://x/ y main", "deb http://x/ y contrib"))
}

func TestAptRepositoryValidate(t *testing.T) {
	m := NewAptRepositoryModule()
	assert.NoError(t, m.Validate(map[string]interface{}{"repo": "deb http://x/ y main"}))
	assert.NoError(t, m.Validate(map[string]interface{}{"repo": "ppa:a/b", "state": "absent"}))
	assert.Error(t, m.Validate(map[string]interface{}{"repo": "http://x/"}))
	assert.Error(t, m.Validate(map[string]interface{}{"repo": "deb http://x/ y main", "filename": "../../etc/passwd"}))
}

func TestAptKeyValidate(t *testing.T) {
	m := NewAptKeyModule()
	assert.NoError(t, m.Validate(map[string]interface{}{"url": "https://x/key.gpg", "keyring": "/etc/apt/keyrings/x.gpg"}))
	assert.NoError(t, m.Validate(map[string]interface{}{"id": "0x9DC858229FC7DD38854AE2D88D81803C0EBFCD88", "state": "absent"}))
	assert.Error(t, m.Validate(map[string]interface{}{"url": "https://x", "data": "k"}))
	assert.Error(t, m.Validate(map[string]interface{}{"keyserver": "keyserver.ubuntu.com", "id": "0EBFCD88"}))
	assert.Error(t, m.Validate(map[string]interface{}{"data": "k", "keyring": "relative.gpg"}))
	assert.Error(t, m.Validate(map[string]interface{}{"state": "absent"}))
}
