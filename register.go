package discord

import (
	"context"

	"github.com/Herrscherd/dctl"
	"github.com/Herrscherd/herrscher-contracts"
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
			Config: []contracts.Setting{
				{Key: "token", Env: "DISCORD_BOT_TOKEN", Help: "Discord bot token", Required: true},
				{Key: "channel", Env: "DISCORD_CHANNEL_ID", Help: "default channel id"},
			},
		},
		Gateway: NewGatewaySet,
	})
}

// NewGatewaySet builds the Discord channel from config: it wires the outbound
// gateway, the read/status reader, the channel admin and the reachability prober.
func NewGatewaySet(ctx context.Context, cfg contracts.PluginConfig) (contracts.GatewaySet, error) {
	c := dctl.New(cfg.Get("token"), cfg.Get("channel"))
	return contracts.GatewaySet{
		Gateway: NewGateway(discordClient{c}),
		Reader:  NewPlatform(c),
		Admin:   NewChannelAdmin(c),
		Prober:  NewProber(c),
	}, nil
}
