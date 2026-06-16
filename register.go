package discord

import (
	"context"

	"github.com/Akayashuu/dctl"
	"github.com/Akayashuu/herrscher-contracts"
)

// init self-registers the Discord gateway into the global plugin registry. A
// blank import of this package (in the host's generated plugins.go) is enough to
// make the gateway discoverable — no wiring in the host. The factory builds its
// own dctl client from config so the plugin stays self-contained.
func init() {
	contracts.Register(contracts.Plugin{
		Manifest: contracts.Manifest{
			Kind:         "discord",
			Category:     contracts.CategoryGateway,
			Capabilities: contracts.Capabilities{Reactions: true, SelectMenus: true, Replies: true},
		},
		Gateway: func(_ context.Context, cfg contracts.PluginConfig) (contracts.Gateway, error) {
			return NewGateway(dctl.New(cfg.Get("token"), cfg.Get("channel"))), nil
		},
	})
}
