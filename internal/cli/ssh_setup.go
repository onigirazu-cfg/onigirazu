package cli

import (
	"fmt"

	"github.com/onigirazu-cfg/onigirazu/internal/config"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
)

// setupSSHFromConfig loads the configuration (--config, or discovered from
// dir) and sets up SSH from it: known_hosts file, strict host keys, timeout.
// Every command that connects to hosts calls it before connecting; without it
// connections use ~/.ssh/known_hosts and ignore the configuration.
func setupSSHFromConfig(dir string) error {
	cfg, err := config.LoadConfigWithDiscovery(configPath, dir)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	sshpkg.InitializeGlobalPoolWithLogger(cfg, logger.New(false))
	return nil
}
