package discord

import (
	"context"
	"testing"

	"github.com/Akayashuu/herrscher-contracts"
)

func TestSelfRegisteredAsGateway(t *testing.T) {
	for _, p := range contracts.Default.Gateways() {
		if p.Manifest.Kind == "discord" {
			if p.Gateway == nil {
				t.Fatal("registered discord plugin has a nil gateway factory")
			}
			g, err := p.Gateway(context.Background(), contracts.PluginConfig{})
			if err != nil || g == nil {
				t.Fatalf("factory failed: g=%v err=%v", g, err)
			}
			return
		}
	}
	t.Fatal("discord gateway did not self-register into contracts.Default")
}
