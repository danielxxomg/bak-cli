package actions

import "github.com/danielxxomg/bak-cli/internal/config"

// loadConfigOr returns the bak-cli config via the injected loader, falling
// back to config.Load when loader is nil. It consolidates the duplicated
// config-loading nil-check previously inlined in PullAction.Run and
// PushAction.shouldEncrypt, keeping error propagation identical to the
// prior inline implementations.
//
// loader is the func-typed ConfigLoader struct field (func() (*config.Config,
// error)), not the ConfigLoader interface in interfaces.go (which returns the
// simplified actions.Config). loadConfigOr stays in internal/actions because
// only pull and push use it and internal/backup cannot import internal/actions.
func loadConfigOr(loader func() (*config.Config, error)) (*config.Config, error) {
	if loader != nil {
		return loader()
	}
	return config.Load()
}
