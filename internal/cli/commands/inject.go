package commands

import "github.com/yourusername/gollm/internal/config"

// injectedConfig allows tests to inject a configuration without touching disk.
var injectedConfig *config.Config

// SetInjectedConfig sets the in-memory configuration used by commands during tests.
func SetInjectedConfig(cfg *config.Config) { injectedConfig = cfg }

func getInjectedOrLoad() (*config.Config, error) {
	if injectedConfig != nil {
		return injectedConfig, nil
	}
	return config.Load()
}
